// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package signing

import (
	"bytes"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"
)

// computeHashStr generates a hash from data object(s) and generates a string
// to be stored in the signature block
func computeHashStr(fimg *sif.FileImage, descr []*sif.Descriptor) string {
	hash := sha512.New384()
	for _, v := range descr {
		hash.Write(v.GetData(fimg))
	}

	sum := hash.Sum(nil)

	return fmt.Sprintf("SIFHASH:\n%x", sum)
}

// sifAddSignature adds a signature block to a SIF file
func sifAddSignature(fimg *sif.FileImage, groupid, link uint32, fingerprint [20]byte, signature []byte) error {
	// data we need to create a signature descriptor
	siginput := sif.DescriptorInput{
		Datatype: sif.DataSignature,
		Groupid:  groupid,
		Link:     link,
		Fname:    "part-signature",
		Data:     signature,
	}
	siginput.Size = int64(binary.Size(siginput.Data))

	// extra data needed for the creation of a signature descriptor
	err := siginput.SetSignExtra(sif.HashSHA384, hex.EncodeToString(fingerprint[:]))
	if err != nil {
		return err
	}

	// add new signature data object to SIF file
	err = fimg.AddObject(siginput)
	if err != nil {
		return err
	}

	return nil
}

// descrToSign determines via argument or interactively which descriptor to sign
func descrToSign(fimg *sif.FileImage, id uint32, isGroup bool) (descr []*sif.Descriptor, err error) {
	descr = make([]*sif.Descriptor, 1)

	if id == 0 {
		descr[0], _, err = fimg.GetPartPrimSys()
		if err != nil {
			return nil, fmt.Errorf("no primary partition found")
		}
	} else if isGroup {
		var search = sif.Descriptor{
			Groupid: id | sif.DescrGroupMask,
		}
		descr, _, err = fimg.GetFromDescr(search)
		if err != nil {
			return nil, fmt.Errorf("no descriptors found for groupid %v", id)
		}
	} else {
		descr[0], _, err = fimg.GetFromDescrID(id)
		if err != nil {
			return nil, fmt.Errorf("no descriptor found for id %v", id)
		}
	}

	return
}

// Sign takes the path of a container and generates an OpenPGP signature block for
// its system partition. Sign uses the private keys found in the default
// location if available or helps the user by prompting with key generation
// configuration options. In its current form, Sign also pushes, when desired,
// public material to a key server.
func Sign(cpath, url string, id uint32, isGroup bool, keyIdx int, authToken string) error {
	elist, err := sypgp.LoadPrivKeyring()
	if err != nil {
		return fmt.Errorf("could not load private keyring: %s", err)
	}

	// Generate a private key usable for signing
	var entity *openpgp.Entity
	if elist == nil {
		resp, err := sypgp.AskQuestion("No OpenPGP signing keys found, autogenerate? [Y/n] ")
		if err != nil {
			return fmt.Errorf("could not read response: %s", err)
		}
		if resp == "" || resp == "y" || resp == "Y" {
			entity, err = sypgp.GenKeyPair()
			if err != nil {
				return fmt.Errorf("generating openpgp key pair failed: %s", err)
			}
		} else {
			return fmt.Errorf("cannot sign without installed keys")
		}
		resp, err = sypgp.AskQuestion("Upload public key %X to %s? [Y/n] ", entity.PrimaryKey.Fingerprint, url)
		if err != nil {
			return err
		}
		if resp == "" || resp == "y" || resp == "Y" {
			if err = sypgp.PushPubkey(entity, url, authToken); err != nil {
				return fmt.Errorf("failed while pushing public key to server: %s", err)
			}
			fmt.Printf("Uploaded key successfully!\n")
		}
	} else {
		if keyIdx != -1 { // -k <idx> has been specified
			if keyIdx >= 0 && keyIdx < len(elist) {
				entity = elist[keyIdx]
			} else {
				return fmt.Errorf("specified (-k, --keyidx) key index out of range")
			}
		} else if len(elist) > 1 {
			entity, err = sypgp.SelectPrivKey(elist)
			if err != nil {
				return fmt.Errorf("failed while reading selection: %s", err)
			}
		} else {
			entity = elist[0]
		}
	}

	// Decrypt key if needed
	if err = sypgp.DecryptKey(entity); err != nil {
		return fmt.Errorf("could not decrypt private key, wrong password?")
	}

	// load the container
	fimg, err := sif.LoadContainer(cpath, false)
	if err != nil {
		return fmt.Errorf("failed to load SIF container file: %s", err)
	}
	defer fimg.UnloadContainer()

	// figure out which descriptor has data to sign
	descr, err := descrToSign(&fimg, id, isGroup)
	if err != nil {
		return fmt.Errorf("signing requires a primary partition: %s", err)
	}

	// signature also include data integrity check
	sifhash := computeHashStr(&fimg, descr)

	// create an ascii armored signature block
	var signedmsg bytes.Buffer
	plaintext, err := clearsign.Encode(&signedmsg, entity.PrivateKey, nil)
	if err != nil {
		return fmt.Errorf("could not build a signature block: %s", err)
	}
	_, err = plaintext.Write([]byte(sifhash))
	if err != nil {
		return fmt.Errorf("failed writing hash value to signature block: %s", err)
	}
	if err = plaintext.Close(); err != nil {
		return fmt.Errorf("I/O error while wrapping up signature block: %s", err)
	}

	// finally add the signature block (for descr) as a new SIF data object
	var groupid, link uint32
	if isGroup {
		groupid = sif.DescrUnusedGroup
		link = descr[0].Groupid
	} else {
		groupid = descr[0].Groupid
		link = descr[0].ID
	}
	err = sifAddSignature(&fimg, groupid, link, entity.PrimaryKey.Fingerprint, signedmsg.Bytes())
	if err != nil {
		return fmt.Errorf("failed adding signature block to SIF container file: %s", err)
	}

	return nil
}

// return all signatures for the primary partition
func getSigsPrimPart(fimg *sif.FileImage) (sigs []*sif.Descriptor, descr []*sif.Descriptor, err error) {
	descr = make([]*sif.Descriptor, 1)

	descr[0], _, err = fimg.GetPartPrimSys()
	if err != nil {
		return nil, nil, fmt.Errorf("no primary partition found")
	}

	sigs, _, err = fimg.GetFromLinkedDescr(descr[0].ID)
	if err != nil {
		return nil, nil, fmt.Errorf("no signatures found for system partition")
	}

	return
}

// return all signatures for specified descriptor
func getSigsDescr(fimg *sif.FileImage, id uint32) (sigs []*sif.Descriptor, descr []*sif.Descriptor, err error) {
	descr = make([]*sif.Descriptor, 1)

	descr[0], _, err = fimg.GetFromDescrID(id)
	if err != nil {
		return nil, nil, fmt.Errorf("no descriptor found for id %v", id)
	}

	sigs, _, err = fimg.GetFromLinkedDescr(id)
	if err != nil {
		return nil, nil, fmt.Errorf("no signatures found for id %v", id)
	}

	return
}

// return all signatures for specified group
func getSigsGroup(fimg *sif.FileImage, id uint32) (sigs []*sif.Descriptor, descr []*sif.Descriptor, err error) {
	// find descriptors that are part of a signing group
	search := sif.Descriptor{
		Groupid: id | sif.DescrGroupMask,
	}
	descr, _, err = fimg.GetFromDescr(search)
	if err != nil {
		return nil, nil, fmt.Errorf("no descriptors found for groupid %v", id)
	}

	// find signature blocks pointing to specified group
	search = sif.Descriptor{
		Datatype: sif.DataSignature,
		Link:     id | sif.DescrGroupMask,
	}
	sigs, _, err = fimg.GetFromDescr(search)
	if err != nil {
		return nil, nil, fmt.Errorf("no signatures found for groupid %v", id)
	}

	return
}

// return all signatures for "id" being unique or group id
func getSigsForSelection(fimg *sif.FileImage, id uint32, isGroup bool) (sigs []*sif.Descriptor, descr []*sif.Descriptor, err error) {
	if id == 0 {
		return getSigsPrimPart(fimg)
	} else if isGroup {
		return getSigsGroup(fimg, id)
	}
	return getSigsDescr(fimg, id)
}

// IsSigned : will return false if the givin container (cpath) is not signed.
// Likewise, will return true if the container is signed and print who signed
// it. will return a error if one occures.
func IsSigned(cpath, url string, id uint32, isGroup bool, authToken string, noPrompt bool) (bool, error) {
	err := Verify(cpath, url, id, isGroup, authToken, noPrompt)
	if err != nil {
		return false, fmt.Errorf("%v", err)
	}
	return true, nil
}

// Verify takes a container path and look for a verification block for a
// specified descriptor. If found, the signature block is used to verify the
// partition hash against the signer's version. Verify takes care of looking
// for OpenPGP keys in the default local store or looks it up from a key server
// if access is enabled.
func Verify(cpath, url string, id uint32, isGroup bool, authToken string, noPrompt bool) error {
	fimg, err := sif.LoadContainer(cpath, true)
	if err != nil {
		return fmt.Errorf("failed to load SIF container file: %s", err)
	}
	defer fimg.UnloadContainer()

	// get all signature blocks (signatures) for ID/GroupID selected (descr) from SIF file
	signatures, descr, err := getSigsForSelection(&fimg, id, isGroup)
	if err != nil {
		return fmt.Errorf("error while searching for signature blocks: %s", err)
	}

	// the selected data object is hashed for comparison against signature block's
	sifhash := computeHashStr(&fimg, descr)

	// load the public keys available locally from the cache
	elist, err := sypgp.LoadPubKeyring()
	if err != nil {
		return fmt.Errorf("could not load public keyring: %s", err)
	}

	// compare freshly computed hash with hashes stored in signatures block(s)
	var authok string
	var netlist openpgp.EntityList
	noLocalKey := false
	for _, v := range signatures {
		// Extract hash string from signature block
		data := v.GetData(&fimg)
		block, _ := clearsign.Decode(data)
		if block == nil {
			return fmt.Errorf("failed to parse signature block")
		}

		if !bytes.Equal(bytes.TrimRight(block.Plaintext, "\n"), []byte(sifhash)) {
			sylog.Infof("NOTE: group signatures will fail if new data is added to a group")
			sylog.Infof("after the group signature is created.")
			return fmt.Errorf("hashes differ, data may be corrupted")
		}

		// (1) Data integrity is verified, (2) now validate identify of signers

		// get the entity fingerprint for the signature block
		fingerprint, err := v.GetEntityString()
		if err != nil {
			return fmt.Errorf("could not get the signing entity fingerprint: %s", err)
		}

		// try to verify with local OpenPGP store first
		signer, err := openpgp.CheckDetachedSignature(elist, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body)
		if err != nil {
			// verification with local keyring failed, try to fetch from key server
			sylog.Infof("Image '%v' is signed with key(s) that are not in your keyring.", cpath)
			sylog.Infof("Searching 'https://keys.sylabs.io' for key (0x%X)...", fingerprint[24:])
			netlist, err = sypgp.FetchPubkey(fingerprint, url, authToken, noPrompt)
			if err != nil {
				return fmt.Errorf("could not fetch public key from server: %s", err)
			}
			sylog.Infof("key found.")

			block, _ := clearsign.Decode(data)
			if block == nil {
				return fmt.Errorf("failed to parse signature block")
			}

			// try verification again with downloaded key
			signer, err = openpgp.CheckDetachedSignature(netlist, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body)
			if err != nil {
				return fmt.Errorf("signature verification failed: %s", err)
			}
			noLocalKey = true
		}

		// Get first Identity data for convenience
		var name string
		for _, i := range signer.Identities {
			name = i.Name
			break
		}
		authok += fmt.Sprintf("\t%s, KeyID %X\n", name, signer.PrimaryKey.KeyId)
	}
	sylog.Infof("Container is signed")
	fmt.Printf("Data integrity checked, authentic and signed by:\n")
	fmt.Printf("%v", authok)

	// if theres no local key, then ask to store it
	if noLocalKey {
		if noPrompt {
			// always store key when prompts disabled
			if err = sypgp.StorePubKey(netlist[0]); err != nil {
				return fmt.Errorf("could not store public key: %s", err)
			}
		} else {
			// Ask to store new public key
			resp, err := sypgp.AskQuestion("Would you like to add this key to your local keyring? [Y/n] ")
			if err != nil {
				return err
			}
			if resp == "" || resp == "y" || resp == "Y" {
				if err = sypgp.StorePubKey(netlist[0]); err != nil {
					return fmt.Errorf("could not store public key: %s", err)
				}
			}
		}
	}

	return nil
}

func getSignEntities(fimg *sif.FileImage) ([]string, error) {
	// get all signature blocks (signatures) for ID/GroupID selected (descr) from SIF file
	signatures, _, err := getSigsPrimPart(fimg)
	if err != nil {
		return nil, err
	}

	var entities []string
	for _, v := range signatures {
		fingerprint, err := v.GetEntityString()
		if err != nil {
			return nil, err
		}
		entities = append(entities, fingerprint)
	}

	return entities, nil
}

// GetSignEntities returns all signing entities for an ID/Groupid
func GetSignEntities(cpath string) ([]string, error) {
	fimg, err := sif.LoadContainer(cpath, true)
	if err != nil {
		return nil, err
	}
	defer fimg.UnloadContainer()

	return getSignEntities(&fimg)
}

// GetSignEntitiesFp returns all signing entities for an ID/Groupid
func GetSignEntitiesFp(fp *os.File) ([]string, error) {
	fimg, err := sif.LoadContainerFp(fp, true)
	if err != nil {
		return nil, err
	}

	return getSignEntities(&fimg)
}

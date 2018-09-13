// Copyright (c) 2018, Sylabs Inc. All rights reserved.
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

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/sypgp"

	"github.com/sylabs/sif/pkg/sif"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"
)

// computeHashStr generates a hash from a data object and generates a string
// to be stored in the signature block
func computeHashStr(fimg *sif.FileImage, descr *sif.Descriptor) string {
	sum := sha512.Sum384(fimg.Filedata[descr.Fileoff : descr.Fileoff+descr.Filelen])

	return fmt.Sprintf("SIFHASH:\n%x", sum)
}

// sifAddSignature adds a signature block to a SIF file
func sifAddSignature(fimg *sif.FileImage, descr *sif.Descriptor, fingerprint [20]byte, signature []byte) error {
	// data we need to create a signature descriptor
	siginput := sif.DescriptorInput{
		Datatype: sif.DataSignature,
		Groupid:  descr.Groupid,
		Link:     descr.ID,
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
func descrToSign(fimg *sif.FileImage) (descr *sif.Descriptor, err error) {
	descr, _, err = fimg.GetPartPrimSys()
	if err != nil {
		return
	}

	return
}

// Sign takes the path of a container and generates an OpenPGP signature block for
// its system partition. Sign uses the private keys found in the default
// location if available or helps the user by prompting with key generation
// configuration options. In its current form, Sign also pushes, when desired,
// public material to a key server.
func Sign(cpath, url, authToken string) error {
	elist, err := sypgp.LoadPrivKeyring()
	if err != nil {
		return err
	}

	// Find a generate a private key usable for signing
	var entity *openpgp.Entity
	if elist == nil {
		resp, err := sypgp.AskQuestion("No OpenPGP signing keys found, autogenerate? [Y/n] ")
		if err != nil {
			return err
		}
		if resp == "" || resp == "y" || resp == "Y" {
			entity, err = sypgp.GenKeyPair()
			if err != nil {
				return err
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
				return err
			}
			fmt.Printf("Uploaded key successfully!\n")
		}
	} else {
		if len(elist) > 1 {
			entity, err = sypgp.SelectPrivKey(elist)
			if err != nil {
				return err
			}
		} else {
			entity = elist[0]
		}
	}

	// Decrypt key if needed
	sypgp.DecryptKey(entity)

	// load the container
	fimg, err := sif.LoadContainer(cpath, false)
	if err != nil {
		return err
	}
	defer fimg.UnloadContainer()

	// figure out which descriptor has data to sign
	descr, err := descrToSign(&fimg)
	if err != nil {
		return err
	}

	// signature also include data integrity check
	sifhash := computeHashStr(&fimg, descr)

	// create an ascii armored signature block
	var signedmsg bytes.Buffer
	plaintext, err := clearsign.Encode(&signedmsg, entity.PrivateKey, nil)
	if err != nil {
		return err
	}
	_, err = plaintext.Write([]byte(sifhash))
	if err != nil {
		return err
	}
	if err = plaintext.Close(); err != nil {
		return err
	}

	// finally add the signature block (for descr) as a new SIF data object
	err = sifAddSignature(&fimg, descr, entity.PrimaryKey.Fingerprint, signedmsg.Bytes())
	if err != nil {
		return err
	}

	return nil
}

// return all signatures for the primary partition
func getSigsPrimPart(fimg *sif.FileImage) (sigs []*sif.Descriptor, descr *sif.Descriptor, err error) {
	descr, _, err = fimg.GetPartPrimSys()
	if err != nil {
		return
	}

	sigs, _, err = fimg.GetFromLinkedDescr(descr.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("no signature found for system partition: %s", err)
	}

	return
}

// return all signatures for "id" being unique or group id
func getSigsForSelection(fimg *sif.FileImage) (sigs []*sif.Descriptor, descr *sif.Descriptor, err error) {
	return getSigsPrimPart(fimg)
}

// Verify takes a container path and look for a verification block for a
// specified descriptor. If found, the signature block is used to verify the
// partition hash against the signer's version. Verify takes care of looking
// for OpenPGP keys in the default local store or looks it up from a key server
// if access is enabled.
func Verify(cpath, url, authToken string) error {
	fimg, err := sif.LoadContainer(cpath, true)
	if err != nil {
		return err
	}
	defer fimg.UnloadContainer()

	// get all signature blocks (signatures) for ID/GroupID selected (descr) from SIF file
	signatures, descr, err := getSigsForSelection(&fimg)
	if err != nil {
		return err
	}

	// the selected data object is hashed for comparison against signature block's
	sifhash := computeHashStr(&fimg, descr)

	// load the public keys available locally from the cache
	elist, err := sypgp.LoadPubKeyring()
	if err != nil {
		return err
	}

	// compare freshly computed hash with hashes stored in signatures block(s)
	var authok string
	for _, v := range signatures {
		// Extract hash string from signature block
		data := v.GetData(&fimg)
		block, _ := clearsign.Decode(data)
		if block == nil {
			return fmt.Errorf("failed to decode clearsign message")
		}

		if !bytes.Equal(bytes.TrimRight(block.Plaintext, "\n"), []byte(sifhash)) {
			return fmt.Errorf("hash check failed, data or signature block corrupted")
		}

		// (1) Data integrity is verified, (2) now validate identify of signers

		// get the entity fingerprint for the signature block
		fingerprint, err := v.GetEntityString()
		if err != nil {
			return err
		}

		// try to verify with local OpenPGP store first
		signer, err := openpgp.CheckDetachedSignature(elist, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body)
		if err != nil {
			// verification with local keyring failed, try to fetch from key server
			sylog.Infof("key missing, searching key server for KeyID: %s...", fingerprint[24:])
			netlist, err := sypgp.FetchPubkey(fingerprint, url, authToken)
			if err != nil {
				return err
			}
			sylog.Infof("key retreived successfully!")

			block, _ := clearsign.Decode(data)
			if block == nil {
				return fmt.Errorf("failed to decode clearsign message")
			}

			// try verification again with downloaded key
			signer, err = openpgp.CheckDetachedSignature(netlist, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body)
			if err != nil {
				return fmt.Errorf("signature verification failed: %s", err)
			}

			// Ask to store new public key
			resp, err := sypgp.AskQuestion("Store new public key %X? [Y/n] ", signer.PrimaryKey.Fingerprint)
			if err != nil {
				return err
			}
			if resp == "" || resp == "y" || resp == "Y" {
				if err = sypgp.StorePubKey(netlist[0]); err != nil {
					return err
				}
			}
		}

		// Get first Identity data for convenience
		var name string
		for _, i := range signer.Identities {
			name = i.Name
			break
		}
		authok += fmt.Sprintf("\t%s, KeyID %X\n", name, signer.PrimaryKey.KeyId)
	}
	fmt.Printf("Data integrity checked, authentic and signed by:\n")
	fmt.Print(authok)

	return nil
}

// GetSignEntities returns all signing entities for an ID/Groupid
func GetSignEntities(cpath string) ([]string, error) {
	fimg, err := sif.LoadContainer(cpath, true)
	if err != nil {
		return nil, err
	}
	defer fimg.UnloadContainer()

	// get all signature blocks (signatures) for ID/GroupID selected (descr) from SIF file
	signatures, _, err := getSigsPrimPart(&fimg)
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

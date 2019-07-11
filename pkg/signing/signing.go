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
	"errors"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"
)

// ErrVerificationFail is the error when the verify fails
var ErrVerificationFail = errors.New("verification failed")

var errNotFound = errors.New("key does not exist in local, or remote keystore")
var errNotFoundLocal = errors.New("key not in local keyring")

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
// location.
func Sign(cpath string, id uint32, isGroup bool, keyIdx int) error {
	keyring := sypgp.NewHandle("")

	elist, err := keyring.LoadPrivKeyring()
	if err != nil {
		return fmt.Errorf("could not load private keyring: %s", err)
	}

	// Generate a private key usable for signing
	var entity *openpgp.Entity
	if elist == nil {
		return fmt.Errorf("no private keys in keyring. use 'key newpair' to generate a key, or 'key import' to import a private key from gpg")
	}
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

	// Decrypt key if needed
	if err = sypgp.DecryptKey(entity, ""); err != nil {
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

// IsSigned Takse a container path (cpath), and will verify that
// container. Returns false if the container is not signed, likewise,
// will return true if the container is signed. Also returns a error
// if one occures, eg. "the container is not signed", or "container is
// signed by a unknown signer".
func IsSigned(cpath, keyServerURI string, id uint32, isGroup bool, authToken string, noPrompt bool) (bool, error) {
	noLocalKey, err := Verify(cpath, keyServerURI, id, isGroup, authToken, false, noPrompt, true)
	if err != nil {
		return false, fmt.Errorf("unable to verify container: %s", cpath)
	}
	if noLocalKey {
		sylog.Warningf("Container might not be trusted; run 'singularity verify %s' to show who signed it", cpath)
	} else {
		sylog.Infof("Container is trusted - run 'singularity key list' to list your trusted keys")
	}
	return true, nil
}

// Verify takes a container path (cpath), and look for a verification block
// for a specified descriptor. If found, the signature block is used to verify
// the partition hash against the signer's version. Verify will look for OpenPGP
// keys in the default local keyring, if non is found, it will then looks it up
// from a key server if access is enabled, or if localVerify is false. Returns
// true, if theres no local key matching a signers entity.
func Verify(cpath, keyServiceURI string, id uint32, isGroup bool, authToken string, localVerify, noPrompt, quiet bool) (bool, error) {
	keyring := sypgp.NewHandle("")

	notLocalKey := false

	fimg, err := sif.LoadContainer(cpath, true)
	if err != nil {
		return false, fmt.Errorf("failed to load SIF container file: %s", err)
	}
	defer fimg.UnloadContainer()

	// get all signature blocks (signatures) for ID/GroupID selected (descr) from SIF file
	signatures, descr, err := getSigsForSelection(&fimg, id, isGroup)
	if err != nil {
		return false, fmt.Errorf("error while searching for signature blocks: %s", err)
	}

	// the selected data object is hashed for comparison against signature block's
	sifhash := computeHashStr(&fimg, descr)

	// setup some colors
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	var fail bool
	var errRet error
	var author string

	author += fmt.Sprintf("Container is signed by %d key(s):\n\n", len(signatures))
	// compare freshly computed hash with hashes stored in signatures block(s)
	for _, v := range signatures {
		// get the entity fingerprint for the signature block
		fingerprint, err := v.GetEntityString()
		if err != nil {
			sylog.Errorf("could not get the signing entity fingerprint from partition ID: %d: %s", v.ID, err)
			fail = true
			continue
		}
		author += fmt.Sprintf("Verifying signature F: %s:\n", fingerprint)

		// Extract hash string from signature block
		data := v.GetData(&fimg)
		block, _ := clearsign.Decode(data)
		if block == nil {
			sylog.Verbosef("%s signature key (%s) corrupted, unable to read data", red("error:"), fingerprint)
			author += fmt.Sprintf("%-18s Signature corrupted, unable to read data\n\n", red("[FAIL]"))
			fail = true
			continue
		}

		// (1) try to get identity of signer
		i, local, err := getSignerIdentity(keyring, v, block, data, fingerprint, keyServiceURI, authToken, noPrompt, localVerify)
		if err != nil {
			// use [MISSING] if we get an error we expect
			if err == errNotFound || err == errNotFoundLocal {
				author += fmt.Sprintf("%-18s %s\n", red("[MISSING]"), err)
			} else {
				author += fmt.Sprintf("%-18s %s\n", red("[FAIL]"), err)
			}
			fail = true
		} else {
			prefix := green("[LOCAL]")
			if !local {
				prefix = yellow("[REMOTE]")
				notLocalKey = true
			}

			author += fmt.Sprintf("%-18s %s\n", prefix, i)
		}

		// (2) Verify data integrity by comparing hashes
		if !bytes.Equal(bytes.TrimRight(block.Plaintext, "\n"), []byte(sifhash)) {
			sylog.Verbosef("%s key (%s) hash differs, data may be corrupted", red("error:"), fingerprint)
			author += fmt.Sprintf("%-18s system partition hash differs, data may be corrupted\n", red("[FAIL]"))
			fail = true
		} else {
			author += fmt.Sprintf("%-18s Data integrity verified\n", green("[OK]"))
		}
		author += fmt.Sprintf("\n")
	}

	if !quiet {
		fmt.Printf("%s", author)
	}

	if fail {
		errRet = ErrVerificationFail
	}

	return notLocalKey, errRet
}

// Get first Identity data for convenience
func getFirstIdentity(e *openpgp.Entity) string {
	for _, i := range e.Identities {
		return i.Name
	}
	return ""
}

func getSignerIdentity(keyring *sypgp.Handle, v *sif.Descriptor, block *clearsign.Block, data []byte, fingerprint, keyServiceURI, authToken string, noPrompt, local bool) (string, bool, error) {
	// load the public keys available locally from the cache
	elist, err := keyring.LoadPubKeyring()
	if err != nil {
		return "", false, fmt.Errorf("could not load public keyring: %s", err)
	}

	// search local keyring for key that matches signature first
	signer, err := openpgp.CheckDetachedSignature(elist, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body)
	if err == nil {
		return getFirstIdentity(signer), true, nil
	}

	// if theres a error, thats probably because we dont have a local key. So download it and try again
	// skip downloading and say we failed
	if local {
		return "", false, errNotFoundLocal
	}

	// this is needed to reset the block objects reader since it is consumed in the last call
	block, _ = clearsign.Decode(data)
	if block == nil {
		return "", false, fmt.Errorf("failed to parse signature block")
	}

	// download the key
	sylog.Verbosef("Key not found in local keyring, checking remote keystore: %s\n", fingerprint[32:])
	netlist, err := sypgp.FetchPubkey(fingerprint, keyServiceURI, authToken, noPrompt)
	if err != nil {
		return "", false, errNotFound
	}

	sylog.Verbosef("Found key in remote keystore: %s", fingerprint[32:])
	// search remote keyring for key that matches signature
	signer, err = openpgp.CheckDetachedSignature(netlist, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body)
	if err == nil {
		return getFirstIdentity(signer), false, nil
	}

	return "", false, err
}

func getSignEntities(fimg *sif.FileImage) ([]string, error) {
	// get all signature blocks (signatures) for ID/GroupID selected (descr) from SIF file
	signatures, _, err := getSigsPrimPart(fimg)
	if err != nil {
		return nil, err
	}

	entities := make([]string, 0, len(signatures))
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

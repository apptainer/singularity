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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

// Key is for json formatting.
type Key struct {
	Signer KeyEntity
}

// KeyEntity holds all the key info, used for json output.
type KeyEntity struct {
	Name        string
	Fingerprint string
	KeyLocal    bool
	KeyCheck    bool
	DataCheck   bool
}

// KeyList is a list of one or more keys.
type KeyList struct {
	Signatures int
	SignerKeys []*Key
}

// computeHashStr generates a hash from data object(s) and generates a string
// to be stored in the signature block
func computeHashStr(fimg *sif.FileImage, descr *sif.Descriptor) string {
	hash := sha512.New384()
	hash.Write(descr.GetData(fimg))

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

// Copy-paste from sylabs/sif
// datatypeStr returns a string representation of a datatype.
func datatypeStr(dtype sif.Datatype) string {
	switch dtype {
	case sif.DataDeffile:
		return "Def.FILE"
	case sif.DataEnvVar:
		return "Env.Vars"
	case sif.DataLabels:
		return "JSON.Labels"
	case sif.DataPartition:
		return "FS"
	case sif.DataSignature:
		return "Signature"
	case sif.DataGenericJSON:
		return "JSON.Generic"
	case sif.DataGeneric:
		return "Generic/Raw"
	}
	return "Unknown data-type"
}

func getDataPartitionToSign(fimg *sif.FileImage, dataType sif.Datatype) ([]*sif.Descriptor, error) {
	sylog.Debugf("Looking for: %s partition to sign...", datatypeStr(dataType))
	// We are using ID 0 (skipping ID), because we are looking for all Datatypes,
	// and ID's will limit the search.
	data, _, err := fimg.GetLinkedDescrsByType(uint32(0), dataType)
	if err != nil && err != sif.ErrNotFound {
		return nil, fmt.Errorf("failed to get descr for deffile: %s", err)
	}
	sylog.Debugf("Found %d partitions", len(data))

	return data, nil
}

// descrToSign determines via argument or interactively which descriptor to sign
func descrToSign(fimg *sif.FileImage, id uint32, isGroup bool) ([]*sif.Descriptor, error) {
	descr := make([]*sif.Descriptor, 1)
	var err error

	if id == 0 {
		descr[0], _, err = fimg.GetPartPrimSys()
		if err != nil {
			return nil, fmt.Errorf("no primary partition found")
		}

		// signableDatatypes is a list of all the signable Datatypes, all
		// but DataSignature, since theres no need to sign a signature.
		signableDatatypes := []sif.Datatype{
			sif.DataDeffile, sif.DataEnvVar,
			sif.DataLabels, sif.DataGenericJSON,
			sif.DataGeneric, sif.DataCryptoMessage,
		}

		for _, datatype := range signableDatatypes {
			data, err := getDataPartitionToSign(fimg, datatype)
			if err != nil {
				return nil, err
			}
			descr = append(descr, data...)
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

	return descr, nil
}

// Sign takes the path of a container and generates an OpenPGP signature block for
// its system partition. Sign uses the private keys found in the default
// location.
func Sign(cpath string, id uint32, isGroup bool, keyIdx int) error {
	keyring := sypgp.NewHandle("")

	// Load a private key usable for signing
	elist, err := keyring.LoadPrivKeyring()
	if err != nil {
		return fmt.Errorf("could not load private keyring: %s", err)
	}
	if elist == nil {
		return fmt.Errorf("no private keys in keyring. use 'key newpair' to generate a key, or 'key import' to import a private key from gpg")
	}

	var entity *openpgp.Entity
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
	if entity.PrivateKey.Encrypted {
		sylog.Debugf("Decrypting key...")
		if err = sypgp.DecryptKey(entity, ""); err != nil {
			return fmt.Errorf("could not decrypt private key, wrong password?")
		}
	}

	// load the container
	fimg, err := sif.LoadContainer(cpath, false)
	if err != nil {
		return fmt.Errorf("failed to load sif container file: %s", err)
	}
	defer fimg.UnloadContainer()

	// figure out which descriptor has data to sign
	descr, err := descrToSign(&fimg, id, isGroup)
	if err != nil {
		return fmt.Errorf("unable to find a signable partition: %s", err)
	}

	for _, de := range descr {
		sylog.Debugf("Signing %s partition...", datatypeStr(de.Datatype))

		// signature also include data integrity check
		sifhash := computeHashStr(&fimg, de)
		sylog.Debugf("Signing hash: %s\n", sifhash)

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
			link = de.Groupid
		} else {
			groupid = de.Groupid
			link = de.ID
		}
		err = sifAddSignature(&fimg, groupid, link, entity.PrimaryKey.Fingerprint, signedmsg.Bytes())
		if err != nil {
			return fmt.Errorf("failed adding signature block to SIF container file: %s", err)
		}
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

	sigs, _, err = fimg.GetLinkedDescrsByType(descr[0].ID, sif.DataSignature)
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

	sigs, _, err = fimg.GetLinkedDescrsByType(id, sif.DataSignature)
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
func IsSigned(cpath, keyServerURI string, id uint32, isGroup bool, authToken string) (bool, error) {
	_, noLocalKey, err := Verify(cpath, keyServerURI, id, isGroup, authToken, false, false)
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
// a string of formatted output, or json (if jsonVerify is true), and true, if
// theres no local key matching a signers entity.
func Verify(cpath, keyServiceURI string, id uint32, isGroup bool, authToken string, localVerify, jsonVerify bool) (string, bool, error) {
	keyring := sypgp.NewHandle("")

	notLocalKey := false

	fimg, err := sif.LoadContainer(cpath, true)
	if err != nil {
		return "", false, fmt.Errorf("failed to load SIF container file: %s", err)
	}
	defer fimg.UnloadContainer()

	// get all signature blocks (signatures) for ID/GroupID selected (descr) from SIF file
	signatures, descr, err := getSigsForSelection(&fimg, id, isGroup)
	if err != nil {
		return "", false, fmt.Errorf("error while searching for signature blocks: %s", err)
	}

	// the selected data object is hashed for comparison against signature block's
	sifhash := computeHashStr(&fimg, descr[0])

	sylog.Debugf("Verifying hash: %s\n", sifhash)

	// setup some colors
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	var fail bool
	var errRet error
	var author string

	var keySigner *Key
	keyEntityList := KeyList{}

	author += fmt.Sprintf("Container is signed by %d key(s):\n\n", len(signatures))
	// compare freshly computed hash with hashes stored in signatures block(s)
	for _, v := range signatures {
		dataCheck := true
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

			keySigner = makeKeyEntity("", fingerprint, false, false, false)
			keyEntityList.SignerKeys = append(keyEntityList.SignerKeys, keySigner)

			fail = true
			continue
		}

		// (1) try to get identity of signer
		i, local, err := getSignerIdentity(keyring, v, block, data, fingerprint, keyServiceURI, authToken, localVerify)
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
			dataCheck = false
			fail = true
		} else {
			author += fmt.Sprintf("%-18s Data integrity verified\n", green("[OK]"))
		}
		author += fmt.Sprintf("\n")

		keySigner = makeKeyEntity(i, fingerprint, local, true, dataCheck)
		keyEntityList.SignerKeys = append(keyEntityList.SignerKeys, keySigner)
	}
	keyEntityList.Signatures = len(signatures)

	if jsonVerify {
		jsonData, err := json.MarshalIndent(keyEntityList, "", "  ")
		if err != nil {
			return "", notLocalKey, fmt.Errorf("unable to parse json: %s", err)
		}
		author = string(jsonData) + "\n"
	}

	if fail {
		errRet = ErrVerificationFail
	}

	return author, notLocalKey, errRet
}

func makeKeyEntity(name, fingerprint string, local, corrupted, dataCheck bool) *Key {
	if name == "" {
		name = "unknown"
	}

	keySigner := &Key{
		Signer: KeyEntity{
			Name:        name,
			Fingerprint: fingerprint,
			KeyLocal:    local,
			KeyCheck:    corrupted,
			DataCheck:   dataCheck,
		},
	}

	return keySigner
}

// Get first Identity data for convenience
func getFirstIdentity(e *openpgp.Entity) string {
	for _, i := range e.Identities {
		return i.Name
	}
	return ""
}

func getSignerIdentity(keyring *sypgp.Handle, v *sif.Descriptor, block *clearsign.Block, data []byte, fingerprint, keyServiceURI, authToken string, local bool) (string, bool, error) {
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
	netlist, err := sypgp.FetchPubkey(http.DefaultClient, fingerprint, keyServiceURI, authToken, true)
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
		fimg.UnloadContainer()
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

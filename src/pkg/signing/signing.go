// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package signing

import (
	"bytes"
	"crypto/sha512"
	"encoding/binary"
	"fmt"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/sypgp"

	"github.com/sylabs/sif/pkg/sif"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"
)

const (
	keyserverURI = "https://keys.sylabs.io:11371"
)

func sifDataObjectHash(fimg *sif.FileImage) (*bytes.Buffer, error) {
	var msg = new(bytes.Buffer)

	part, _, err := fimg.GetPartFromGroup(sif.DescrDefaultGroup)
	if err != nil {
		sylog.Errorf("%s\n", err)
		return nil, err
	}

	sum := sha512.Sum384(fimg.Filedata[part.Fileoff : part.Fileoff+part.Filelen])

	fmt.Fprintf(msg, "SIFHASH:\n%x", sum)

	return msg, nil
}

func sifAddSignature(fingerprint [20]byte, fimg *sif.FileImage, signature []byte) error {
	part, _, err := fimg.GetPartFromGroup(sif.DescrDefaultGroup)
	if err != nil {
		sylog.Errorf("%s\n", err)
		return err
	}

	// data we need to create a signature descriptor
	siginput := sif.DescriptorInput{
		Datatype: sif.DataSignature,
		Groupid:  sif.DescrDefaultGroup,
		Link:     part.ID,
		Fname:    "part-signature",
		Data:     signature,
	}
	siginput.Size = int64(binary.Size(siginput.Data))

	// extra data needed for the creation of a signature descriptor
	siginfo := sif.Signature{
		Hashtype: sif.HashSHA384,
		Entity:   [sif.DescrEntityLen]byte{},
	}
	copy(siginfo.Entity[:], fingerprint[:])

	// serialize the signature data for integration with the base descriptor input
	if err := binary.Write(&siginput.Extra, binary.LittleEndian, siginfo); err != nil {
		return fmt.Errorf("binary encoding siginput.Extra: %s", err)
	}

	// add new signature data object to SIF file
	if err = fimg.AddObject(siginput); err != nil {
		return fmt.Errorf("adding new signature to SIF file: %s", err)
	}

	return nil
}

// Sign takes the path of a container and generates a PGP signature block for
// its system partition. Sign uses the private keys found in the default
// location if available or helps the user by prompting with key generation
// configuration options. In its current form, Sign also pushes public material
// to a key server if enabled. This should be a separate step in the next round
// of development.
func Sign(cpath, authToken string) error {
	var el openpgp.EntityList
	var en *openpgp.Entity
	var err error

	if el, err = sypgp.LoadPrivKeyring(); err != nil {
		return err
	} else if el == nil {
		fmt.Println("No Private Keys found in SYPGP store, generating RSA pair for you.")
		err = sypgp.GenKeyPair()
		if err != nil {
			return err
		}
		if el, err = sypgp.LoadPrivKeyring(); err != nil || el == nil {
			return err
		}
		fmt.Printf("Sending PGP public key material: %0X => %s.\n", el[0].PrimaryKey.Fingerprint, keyserverURI)
		err = sypgp.PushPubkey(el[0], keyserverURI, authToken)
		if err != nil {
			return err
		}
	}

	if len(el) > 1 {
		if en, err = sypgp.SelectPrivKey(el); err != nil {
			return err
		}
	} else {
		en = el[0]
	}
	sypgp.DecryptKey(en)

	// load the container
	fimg, err := sif.LoadContainer(cpath, false)
	if err != nil {
		sylog.Errorf("error loading sif file %s: %s\n", cpath, err)
		return err
	}
	defer fimg.UnloadContainer()

	msg, err := sifDataObjectHash(&fimg)
	if err != nil {
		return err
	}

	var signedmsg bytes.Buffer
	plaintext, err := clearsign.Encode(&signedmsg, en.PrivateKey, nil)
	if err != nil {
		sylog.Errorf("error from Encode: %s\n", err)
		return err
	}
	if _, err = plaintext.Write(msg.Bytes()); err != nil {
		sylog.Errorf("error from Write: %s\n", err)
		return err
	}
	if err = plaintext.Close(); err != nil {
		sylog.Errorf("error from Close: %s\n", err)
		return err
	}

	if err = sifAddSignature(en.PrimaryKey.Fingerprint, &fimg, signedmsg.Bytes()); err != nil {
		return err
	}

	return nil
}

// Verify takes a container path and look for a verification block for a
// system partition. If found, the signature block is used to verify the
// partition hash against the signer's version. Verify takes care of looking
// for PGP keys in the default local store or looks it up from a key server
// if access is enabled.
func Verify(cpath, authToken string) error {
	var el openpgp.EntityList

	// load the container
	fimg, err := sif.LoadContainer(cpath, true)
	if err != nil {
		sylog.Errorf("error loading sif file %s: %s\n", cpath, err)
		return err
	}
	defer fimg.UnloadContainer()

	msg, err := sifDataObjectHash(&fimg)
	if err != nil {
		return err
	}

	part, _, err := fimg.GetPartFromGroup(sif.DescrDefaultGroup)
	if err != nil {
		return fmt.Errorf("no system partition found: %s", err)
	}

	sig, _, err := fimg.GetFromLinkedDescr(part.ID)
	if err != nil {
		return fmt.Errorf("no signature found for system partition: %s", err)
	}

	data := fimg.Filedata[sig.Fileoff : sig.Fileoff+sig.Filelen]

	block, _ := clearsign.Decode(data)
	if block == nil {
		sylog.Errorf("failed to decode clearsign message\n")
		return fmt.Errorf("failed to decode clearsign message")
	}

	if !bytes.Equal(bytes.TrimRight(block.Plaintext, "\n"), msg.Bytes()) {
		sylog.Errorf("Sif hash string mismatch -- don't use:\nsigned:     %s\ncalculated: %s\n", msg.String(), block.Plaintext)
		return fmt.Errorf("Sif hash string mismatch -- don't use")
	}

	if el, err = sypgp.LoadPubKeyring(); err != nil {
		return err
	}

	// get the entity fingerprint for the found signature block
	fingerprint, err := sig.GetEntityString()
	if err != nil {
		return err
	}

	// try to verify with local PGP store first
	var signer *openpgp.Entity
	if signer, err = openpgp.CheckDetachedSignature(el, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body); err != nil {
		sylog.Errorf("failed to check signature: %s\n", err)
		// verification with local keyring failed, try to fetch from key server
		sylog.Infof("Contacting sykeys PGP key management services for: %s\n", fingerprint)
		syel, err := sypgp.FetchPubkey(fingerprint, keyserverURI, authToken)
		if err != nil {
			return err
		}

		block, _ := clearsign.Decode(data)
		if block == nil {
			sylog.Errorf("failed to decode clearsign message\n")
			return fmt.Errorf("failed to decode clearsign message")
		}

		if signer, err = openpgp.CheckDetachedSignature(syel, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body); err != nil {
			sylog.Errorf("failed to check signature: %s\n", err)
			return err
		}
	}
	fmt.Print("Authentic and signed by:\n")
	for _, i := range signer.Identities {
		fmt.Printf("\t%s\n", i.Name)
	}

	return nil
}

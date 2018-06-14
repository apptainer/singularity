// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package signing

import (
	"bytes"
	"crypto/sha512"
	"fmt"

	"github.com/singularityware/singularity/src/pkg/sif"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/sypgp"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"
)

const (
	syKeysAddr = "keys.sylabs.io:11371"
)

func sifDataObjectHash(sinfo *sif.Info) (*bytes.Buffer, error) {
	var msg = new(bytes.Buffer)

	part, err := sif.GetPartition(sinfo, sif.DefaultGroup)
	if err != nil {
		sylog.Errorf("%s\n", err)
		return nil, err
	}

	data, err := sif.CByteRange(sinfo.Mapstart(), part.FileOff(), part.FileLen())
	if err != nil {
		sylog.Errorf("%s\n", err)
		return nil, err
	}
	sum := sha512.Sum384(data)

	fmt.Fprintf(msg, "SIFHASH:\n%x", sum)

	return msg, nil
}

func sifAddSignature(fingerprint [20]byte, sinfo *sif.Info, signature []byte) error {
	var e sif.Eleminfo

	part, err := sif.GetPartition(sinfo, sif.DefaultGroup)
	if err != nil {
		sylog.Errorf("%s\n", err)
		return err
	}

	e.InitSignature(fingerprint, signature, part)

	if err := sif.PutDataObj(&e, sinfo); err != nil {
		sylog.Errorf("%s\n", err)
		return err
	}
	return nil
}

// Sign takes the path of a container and generates a PGP signature block for
// its system partition. Sign uses the private keys found in the default
// location if available or helps the user by prompting with key generation
// configuration options. In its current form, Sign also pushes public material
// to a key server if enabled. This should be a separate step in the next round
// of development.
func Sign(cpath string) error {
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
		fmt.Printf("Sending PGP public key material: %0X => %s.\n", el[0].PrimaryKey.Fingerprint, syKeysAddr)
		err = sypgp.PushPubkey(el[0], syKeysAddr)
		if err != nil {
			return err
		}
	}

	if len(el) > 1 {
		if en, err = sypgp.SelectKey(el); err != nil {
			return err
		}
	} else {
		en = el[0]
	}
	sypgp.DecryptKey(en)

	var sinfo sif.Info
	if err = sif.Load(cpath, &sinfo, 0); err != nil {
		sylog.Errorf("error loading sif file %s: %s\n", cpath, err)
		return err
	}
	defer sif.Unload(&sinfo)

	msg, err := sifDataObjectHash(&sinfo)
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

	if err = sifAddSignature(en.PrimaryKey.Fingerprint, &sinfo, signedmsg.Bytes()); err != nil {
		return err
	}

	return nil
}

// Verify takes a container path and look for a verification block for a
// system partition. If found, the signature block is used to verify the
// partition hash against the signer's version. Verify takes care of looking
// for PGP keys in the default local store or looks it up from a key server
// if access is enabled.
func Verify(cpath string) error {
	var el openpgp.EntityList
	var sinfo sif.Info

	if err := sif.Load(cpath, &sinfo, 0); err != nil {
		sylog.Errorf("%s\n", err)
		return err
	}
	defer sif.Unload(&sinfo)

	msg, err := sifDataObjectHash(&sinfo)
	if err != nil {
		return err
	}

	sig, err := sif.GetSignature(&sinfo)
	if err != nil {
		sylog.Errorf("%s\n", err)
		return err
	}

	data, err := sif.CByteRange(sinfo.Mapstart(), sig.FileOff(), sig.FileLen())
	if err != nil {
		sylog.Errorf("%s\n", err)
		return err
	}

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

	/* try to verify with local PGP store */
	var signer *openpgp.Entity
	if signer, err = openpgp.CheckDetachedSignature(el, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body); err != nil {
		sylog.Errorf("failed to check signature: %s\n", err)
		/* verification with local keyring failed, try to fetch from key server */
		sylog.Infof("Contacting sykeys PGP key management services for: %s\n", sig.GetEntity())
		syel, err := sypgp.FetchPubkey(sig.GetEntity(), syKeysAddr)
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

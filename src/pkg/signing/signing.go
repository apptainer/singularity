// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
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

func computeHashStr(fimg *sif.FileImage, descr *sif.Descriptor) string {
	sum := sha512.Sum384(fimg.Filedata[descr.Fileoff : descr.Fileoff+descr.Filelen])

	return fmt.Sprintf("SIFHASH:\n%x", sum)
}

// adds a signature block for the primary partition
func sifAddSignature(fingerprint [20]byte, fimg *sif.FileImage, signature []byte) error {
	part, _, err := fimg.GetPartPrimSys()
	if err != nil {
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
	err = siginput.SetSignExtra(sif.HashSHA384, hex.EncodeToString(fingerprint[:]))
	if err != nil {
		return err
	}

	// add new signature data object to SIF file
	if err = fimg.AddObject(siginput); err != nil {
		return fmt.Errorf("adding new signature to SIF file: %s", err)
	}

	return nil
}

// Sign takes the path of a container and generates an OpenPGP signature block for
// its system partition. Sign uses the private keys found in the default
// location if available or helps the user by prompting with key generation
// configuration options. In its current form, Sign also pushes public material
// to a key server if enabled. This should be a separate step in the next round
// of development.
func Sign(cpath, url, authToken string) error {
	elist, err := sypgp.LoadPrivKeyring()
	if err != nil {
		return err
	}

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
		resp, err = sypgp.AskQuestion("Upload public key %X to key server? [Y/n] ", entity.PrimaryKey.Fingerprint)
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

	// Decrypt key is needed
	sypgp.DecryptKey(entity)

	// load the container
	fimg, err := sif.LoadContainer(cpath, false)
	if err != nil {
		return err
	}
	defer fimg.UnloadContainer()

	part, _, err := fimg.GetPartPrimSys()
	if err != nil {
		return err
	}

	sifhash := computeHashStr(&fimg, part)
	if err != nil {
		return err
	}

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

	err = sifAddSignature(entity.PrimaryKey.Fingerprint, &fimg, signedmsg.Bytes())
	if err != nil {
		return err
	}

	return nil
}

// XXX: move to SIF/Lookup.go
func getDescrData(fimg *sif.FileImage, descr *sif.Descriptor) []byte {
	return fimg.Filedata[descr.Fileoff : descr.Fileoff+descr.Filelen]
}

// return all signatures for "id" being unique or group id
func getSigsForSelection(fimg *sif.FileImage) (sigs []*sif.Descriptor, descr *sif.Descriptor, err error) {
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

// Verify takes a container path and look for a verification block for a
// system partition. If found, the signature block is used to verify the
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
	if err != nil {
		return err
	}

	// load the public keys available locally from the cache
	elist, err := sypgp.LoadPubKeyring()
	if err != nil {
		return err
	}

	// compare freshly computed hash with hashes stored in signatures block(s)
	for _, v := range signatures {
		// Extract hash string from signature block
		data := getDescrData(&fimg, v)
		block, _ := clearsign.Decode(data)
		if block == nil {
			return fmt.Errorf("failed to decode clearsign message")
		}

		if !bytes.Equal(bytes.TrimRight(block.Plaintext, "\n"), []byte(sifhash)) {
			return fmt.Errorf("hash comparison failed, data or signature block corrupted")
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
			elist, err = sypgp.FetchPubkey(fingerprint, url, authToken)
			if err != nil {
				return err
			}
			sylog.Infof("key retreived successfully!")

			block, _ := clearsign.Decode(data)
			if block == nil {
				return fmt.Errorf("failed to decode clearsign message")
			}

			// try verification again with downloaded key
			signer, err = openpgp.CheckDetachedSignature(elist, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body)
			if err != nil {
				return fmt.Errorf("signature verification failed: %s", err)
			}

			// Ask to store new public key
			resp, err := sypgp.AskQuestion("Store new public key %X? [Y/n] ", signer.PrimaryKey.Fingerprint)
			if err != nil {
				return err
			}
			if resp == "" || resp == "y" || resp == "Y" {
				if err = sypgp.StorePubKey(elist[0]); err != nil {
					return err
				}
			}
		}

		fmt.Printf("Data integrity checked, authentic and signed by:\n")
		// Get first Identity data for convenience
		var name string
		for _, i := range signer.Identities {
			name = i.Name
			break
		}
		fmt.Printf("\t%s, KeyID %X\n", name, signer.PrimaryKey.KeyId)
	}

	return nil
}

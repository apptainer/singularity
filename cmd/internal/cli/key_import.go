// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/errors"
)

func init() {
	KeyImportCmd.Flags().SetInterspersed(false)
}

// KeyImportCmd is `singularity key (or keys) import` and imports a local key into the singularity key store.
var KeyImportCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run:                   importRun,

	Use:     docs.KeyImportUse,
	Short:   docs.KeyImportShort,
	Long:    docs.KeyImportLong,
	Example: docs.KeyImportExample,
}

func doKeyImportCmd(path string) error {
	var fingerprint [20]byte
	// PublicKeyType is the armor type for a PGP public key.
	const PublicKeyType = "PGP PUBLIC KEY BLOCK"

	// PrivateKeyType is the armor type for a PGP private key.
	const PrivateKeyType = "PGP PRIVATE KEY BLOCK"

	// Open the file to check if this corresponds to a private or public key
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// If the block does not correspond to any of public ot private type return error
	block, err := armor.Decode(f)
	if err != nil {
		return err
	}

	if block.Type != PublicKeyType && block.Type != PrivateKeyType {
		return errors.InvalidArgumentError("expected public or private key block, got: " + block.Type)
	}

	// Case on public key import
	if block.Type == PublicKeyType {
		// Load the local public keys as entitylist
		publicEntityList, err := sypgp.LoadPubKeyring()
		if err != nil {
			return err
		}
		// Load the public key as an entitylist
		pathEntityList, err := sypgp.LoadKeyringFromFile(path)
		if err != nil {
			return err
		}
		// Get local keystore (where the key will be stored)
		publicFilePath, err := os.OpenFile(sypgp.PublicPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer publicFilePath.Close()

		// Go through the keyring checking for the given fingerprint
		for _, pathEntity := range pathEntityList {
			isInStore := false
			fingerprint = pathEntity.PrimaryKey.Fingerprint

			for _, publicEntity := range publicEntityList {
				if pathEntity.PrimaryKey.KeyId == publicEntity.PrimaryKey.KeyId {
					isInStore = true
					// Verify that this key has already been added
					break
				}
			}
			if !isInStore {
				if err = pathEntity.Serialize(publicFilePath); err != nil {
					return err
				}
				fmt.Printf("Key with fingerprint %X added succesfully to the keystore\n", fingerprint)
			} else {
				fmt.Printf("The key you want to add with fingerprint %X already belongs to the keystore\n", fingerprint)
			}
		}
	} else {
		// Case on private key import
		if block.Type == PrivateKeyType {

			// Load the local private keys as entitylist
			privateEntityList, err := sypgp.LoadPrivKeyring()
			if err != nil {
				return err
			}
			// Load the private key as an entitylist
			pathEntityList, err := sypgp.LoadKeyringFromFile(path)
			if err != nil {
				return err
			}
			// Get local keyring (where the key will be stored)
			secretFilePath, err := os.OpenFile(sypgp.SecretPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			defer secretFilePath.Close()

			// Go through the keystore checking for the given fingerprint
			for _, pathEntity := range pathEntityList {
				isInStore := false
				fingerprint = pathEntity.PrimaryKey.Fingerprint

				for _, privateEntity := range privateEntityList {
					if privateEntity.PrimaryKey.Fingerprint == fingerprint {
						isInStore = true
						break
					}

				}

				fmt.Println("KEY ENTITY ____ : ", pathEntity.PrivateKey.Encrypted)
				if !isInStore {

					e := &openpgp.Entity{
						PrimaryKey: pathEntity.PrimaryKey,
						PrivateKey: pathEntity.PrivateKey,
						//Identities:  map[string]*Identity, // indexed by Identity.Name
						Identities: pathEntity.Identities, // indexed by Identity.Name
						//Revocations: []*pathEntity.Signature,
						Revocations: pathEntity.Revocations,
						Subkeys:     pathEntity.Subkeys,
					}

					//e := openpgp.Entity{
					//	PrimaryKey: pathEntity.PrimaryKey,
					//	PrivateKey: pathEntity.PrivateKey,
					//	//PrivateKey.Encrypted: pathEntity.PrivateKey.Encrypted,
					//}

					fmt.Printf("DEBUG 1 : %v\n", pathEntity.PrivateKey)

					//if err = pathEntity.Serialize(secretFilePath); err != nil {
					//	return err
					//}

					fmt.Printf("Using default pass: 1234\n")
					err = sypgp.EncryptKey(e, "1234")
					if err != nil {
						return err
					}

					fmt.Printf("EEEEE: %v\n", e.PrivateKey.Encrypted)
					fmt.Printf("ALLLLLL: %+v\n", e)
					fmt.Printf("\n")
					fmt.Printf("ALLLLLL PATHENTITY: %+v\n", pathEntity)

					//if err = pathEntity.Serialize(secretFilePath); err != nil {
					//	return err
					//}

					err = sypgp.StorePrivKey(e)
					if err != nil {
						return err
					}

					//if err = e.SerializePrivate(secretFilePath, nil); err != nil {
					//	return err
					//}
					fmt.Printf("Key with fingerprint %0X added succesfully to the keystore\n", fingerprint)
				} else {
					fmt.Printf("The key you want to add with fingerprint %0X already belongs to the keystore\n", fingerprint)
				}
			}

		}
	}

	return nil
}

func importRun(cmd *cobra.Command, args []string) {

	if err := doKeyImportCmd(args[0]); err != nil {
		sylog.Errorf("key import command failed: %s", err)
		os.Exit(2)
	}

}

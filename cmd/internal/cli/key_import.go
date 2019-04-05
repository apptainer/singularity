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
			//fingerprint = pathEntity.PrimaryKey.Fingerprint

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
				fmt.Printf("Key with fingerprint %X succesfully added to the keyring\n", pathEntity.PrimaryKey.Fingerprint)
			} else {
				fmt.Printf("The key you want to add with fingerprint %X already belongs to the keyring\n", pathEntity.PrimaryKey.Fingerprint)
			}
		}
		// If private key
	} else if block.Type == PrivateKeyType {

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
			//fingerprint = pathEntity.PrimaryKey.Fingerprint

			for _, privateEntity := range privateEntityList {
				if privateEntity.PrimaryKey.Fingerprint == pathEntity.PrimaryKey.Fingerprint {
					isInStore = true
					break
				}

			}

			if !isInStore {
				// Make a clone of the entity
				newEntity := &openpgp.Entity{
					PrimaryKey:  pathEntity.PrimaryKey,
					PrivateKey:  pathEntity.PrivateKey,
					Identities:  pathEntity.Identities,
					Revocations: pathEntity.Revocations,
					Subkeys:     pathEntity.Subkeys,
				}

				// Check if the key is encrypted, if it is, decrypt it
				if pathEntity.PrivateKey.Encrypted {
					fmt.Fprintf(os.Stderr, "Enter your old password for this key:\n")
					err = sypgp.DecryptKey(newEntity)
					if err != nil {
						return err
					}
				}

				// Get a new password for the key
				fmt.Fprintf(os.Stderr, "Enter a new password for this key:\n")
				newPass, err := sypgp.GetPassphrase(3)
				if err != nil {
					return err
				}
				err = sypgp.EncryptKey(newEntity, newPass)
				if err != nil {
					return err
				}

				// Store the private key
				err = sypgp.StorePrivKey(newEntity)
				if err != nil {
					return err
				}
				fmt.Printf("Key with fingerprint %X succesfully added to the keyring\n", pathEntity.PrimaryKey.Fingerprint)
			} else {
				fmt.Printf("The key you want to add with fingerprint %X already belongs to the keyring\n", pathEntity.PrimaryKey.Fingerprint)
			}
		}
	} else {
		// The code should NEVER get here
		return fmt.Errorf("not a private, or public key")
	}

	return nil
}

func importRun(cmd *cobra.Command, args []string) {

	if err := doKeyImportCmd(args[0]); err != nil {
		sylog.Errorf("key import command failed: %s", err)
		os.Exit(2)
	}

}

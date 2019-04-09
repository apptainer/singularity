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
)

var (
	secretExport bool
	foundKey     bool
)

func init() {
	KeyExportCmd.Flags().SetInterspersed(false)

	KeyExportCmd.Flags().BoolVarP(&secretExport, "secret", "s", false, "export a secret key")
}

// KeyExportCmd is `singularity key export` and exports a public or secret
// key from local keyring.
var KeyExportCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run:                   exportRun,

	Use:     docs.KeyExportUse,
	Short:   docs.KeyExportShort,
	Long:    docs.KeyExportLong,
	Example: docs.KeyExportExample,
}

func doKeyExportCmd(secretExport bool, path string) error {
	// describes the path from either the local public keyring or secret local keyring
	var fetchPath string
	//var keyString string

	if secretExport {
		fetchPath = sypgp.SecretPath()
	} else {
		fetchPath = sypgp.PublicPath()
	}

	//	f, err := os.OpenFile(fetchPath, os.O_RDONLY|os.O_CREATE, 0600)
	//	if err != nil {
	//		return fmt.Errorf("unable to open local keyring: %v", err)
	//	}
	//	defer f.Close()

	// read all the local secret keys
	//	localEntityList, err := openpgp.ReadKeyRing(f)
	//	if err != nil {
	//		return fmt.Errorf("unable to list local keyring: %v", err)
	//	}

	//file, err := os.Create(path)
	//if err != nil {
	//	return err
	//}

	if secretExport {
		// Get a key to export
		//		entityToExport, err := sypgp.SelectPrivKey(localEntityList)
		//		if err != nil {
		//			return err
		//		}
		//		err = sypgp.DecryptKey(entityToExport)
		//		if err != nil {
		//			return err
		//		}

		err := sypgp.ExportPrivateKey(path)
		if err != nil {
			return err
		}

		//keyString, err = sypgp.SerializePrivateEntity(entityToExport, openpgp.PrivateKeyType, nil)
		//file.WriteString(keyString)
		//defer file.Close()
		//if err != nil {
		//	return fmt.Errorf("error encoding private key: %v", err)
		//}

		//fmt.Printf("Setting default password: 1234\n")
		//err = sypgp.EncryptKey(entityToExport, "1234")
		//err = sypgp.EncryptKey(keyString, "1234")
		//if err != nil {
		//	return err
		//}

		//		fmt.Printf("Private key with fingerprint %X correctly exported to file: %s\n", entityToExport.PrimaryKey.Fingerprint, path)
	} else {

		f, err := os.OpenFile(fetchPath, os.O_RDONLY|os.O_CREATE, 0600)
		if err != nil {
			return fmt.Errorf("unable to open local keyring: %v", err)
		}
		defer f.Close()

		localEntityList, err := openpgp.ReadKeyRing(f)
		if err != nil {
			return fmt.Errorf("unable to list local keyring: %v", err)
		}

		entityToExport, err := sypgp.SelectPubKey(localEntityList)
		if err != nil {
			return err
		}
		//		keyString, err = sypgp.SerializePublicEntity(entityToExport, openpgp.PublicKeyType)
		//		file.WriteString(keyString)
		//		defer file.Close()
		fmt.Printf("Public key with fingerprint %X correctly exported to file: %s\n", entityToExport.PrimaryKey.Fingerprint, path)

	}

	return nil
}

func exportRun(cmd *cobra.Command, args []string) {
	if secretExport {
		err := sypgp.ExportPrivateKey(args[0])
		if err != nil {
			sylog.Errorf("key export command failed: %s", err)
			os.Exit(10)
		}
	} else {
		err := sypgp.ExportPubKey(args[0])
		if err != nil {
			sylog.Errorf("key export command failed: %s", err)
			os.Exit(10)
		}
	}
}

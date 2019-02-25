// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
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
)

func init() {
	KeyImportCmd.Flags().SetInterspersed(false)

	KeyImportCmd.Flags().StringVarP(&keyLocalFolderPath, "path", "p", defaultLocalKeyStore, "specify the local folder path to the key to be added")
	KeyImportCmd.Flags().StringVarP(&keyFingerprint, "fingerprint", "f", "", "specify the local folder path to the key to be added")
}

// KeyImportCmd is `singularity keys import` and imports a local key into the key store.
var KeyImportCmd = &cobra.Command{
	Args: cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run:                   importRun,
	Use:                   docs.KeyImportUse,
	Short:                 docs.KeyImportShort,
	Long:                  docs.KeyImportLong,
	Example:               docs.KeyImportExample,
}

func doKeyImportCmd(path string, fingerprint string) error {

	var count int

	el, err := sypgp.LoadPubKeyringFromFileAndFingerPrint(path, fingerprint)
	if err != nil {
		return err
	}
	//load the local file passed as argument to a entity list key store
	elstore, err := sypgp.LoadPubKeyringFromFile(path)
	if err != nil {
		return err
	}
	// get local cache (where the key will be stored)
	fp, err := os.OpenFile(sypgp.PublicPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer fp.Close()

	//go through the keystore checking for the given fingerprint
	for _, e := range elstore {
		storeKey := true
		for _, estore := range el {
			if estore.PrimaryKey.KeyId == e.PrimaryKey.KeyId {
				storeKey = true // Verify that entity is in key store file
				break
			}
		}
		if storeKey {
			if err = el.Serialize(fp); err != nil {
				return err
			}
		}
		else{
			fmt.Printf("This fingerprint does not belong to the given keystore file.")
		}
	}

	fmt.Printf("Adding key(s) with fingerprint %s from %s into local cache %s\n", fingerprint, path, sypgp.PublicPath())

	return nil
}

func importRun(cmd *cobra.Command, args []string) {

	if err := doKeyImportCmd(args[0], args[1]); err != nil {
		sylog.Errorf("keys import command failed: %s", err)
		os.Exit(2)
	}

}

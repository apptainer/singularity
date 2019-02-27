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
}

// KeyImportCmd is `singularity key (or keys) import` and imports a local key into the singularity key store.
var KeyImportCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run:                   importRun,
	Use:                   docs.KeyImportUse,
	Short:                 docs.KeyImportShort,
	Long:                  docs.KeyImportLong,
	Example:               docs.KeyImportExample,
}

func doKeyImportCmd(path string) error {
	var fingerprint [20]byte
	//load the local public keys as entitylist
	el, err := sypgp.LoadPubKeyring()
	if err != nil {
		return err
	}
	//load the public key as an entitylist
	elstore, err := sypgp.LoadPubKeyringFromFile(path)
	if err != nil {
		return err
	}
	// get local keystore (where the key will be stored)
	fp, err := os.OpenFile(sypgp.PublicPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer fp.Close()

	//go through the keystore checking for the given fingerprint
	for _, estore := range elstore {
		isInStore := false
		fingerprint = estore.PrimaryKey.Fingerprint
		for _, e := range el {
			if estore.PrimaryKey.KeyId == e.PrimaryKey.KeyId {
				isInStore = true // Verify that this key has already been added
			}
		}
		if !isInStore {
			if err = estore.Serialize(fp); err != nil {
				return err
			}
			fmt.Printf("Key with fingerprint %0X added succesfully to the keystore\n", fingerprint)
		} else {
			fmt.Printf("The key you want to add with fingerprint %0X already belongs to the keystore\n", fingerprint)
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

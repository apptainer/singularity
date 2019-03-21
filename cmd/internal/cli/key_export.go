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

var secretExport bool
var foundKey bool

func init() {
	KeyExportCmd.Flags().SetInterspersed(false)

	KeyExportCmd.Flags().BoolVarP(&secretExport, "secret", "s", false, "fetch a key on local secret keystore and export it")
	KeyExportCmd.Flags().SetAnnotation("secret", "envkey", []string{"SECRET"})
}

// KeyExportCmd is `singularity key (or keys) export` and exports a key from either the public or secret local key store.
var KeyExportCmd = &cobra.Command{
	Args: cobra.ExactArgs(2),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run:                   exportRun,
	Use:                   docs.KeyExportUse,
	Short:                 docs.KeyExportShort,
	Long:                  docs.KeyExportLong,
	Example:               docs.KeyExportExample,
}

func doKeyExportCmd(secretExport bool, fingerprint string, path string) error {

	if secretExport {

		f, err := os.OpenFile(sypgp.SecretPath(), os.O_RDONLY|os.O_CREATE, 0600)
		if err != nil {
			return fmt.Errorf("unable to open local keyring: %v", err)
		}
		defer f.Close()

		// read all the local secret keys
		elist, err := openpgp.ReadKeyRing(f)
		if err != nil {
			return fmt.Errorf("unable to list local keyring: %v", err)
		}

		foundKey = false
		// sort through them, and remove any that match toDelete
		for i := range elist {
			// if the elist[i] dose not match toDelete, then add it to newKeyList
			if fmt.Sprintf("%X", elist[i].PrimaryKey.Fingerprint) == fmt.Sprintf("%X", fingerprint) {
				foundKey = true
				fmt.Printf("found key!\n")
				break

			}
		}

	} else {

		f, err := os.OpenFile(sypgp.PublicPath(), os.O_RDONLY|os.O_CREATE, 0600)
		if err != nil {
			return fmt.Errorf("unable to open local keyring: %v", err)
		}
		defer f.Close()
	}

	return nil
}

func exportRun(cmd *cobra.Command, args []string) {

	if err := doKeyExportCmd(secretExport, args[0], args[1]); err != nil {
		sylog.Errorf("key export command failed: %s", err)
		os.Exit(2)
	}

}

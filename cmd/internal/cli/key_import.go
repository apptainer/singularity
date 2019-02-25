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

// KeyImportCmd is `singularity key import` and imports a local key into the key store.
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
	fmt.Printf("Adding key %s from %s into the Singularity keystore %s\n", fingerprint, path, sypgp.PublicPath())

	return nil
}

func importRun(cmd *cobra.Command, args []string) {
	if err := doKeyImportCmd(args[0], keyLocalFolderPath); err != nil {
		sylog.Errorf("import failed: %s", err)
		os.Exit(2)
	}
}

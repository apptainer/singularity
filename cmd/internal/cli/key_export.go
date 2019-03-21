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
)

var secretExport bool

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
		fmt.Printf("secret key export\n")
	} else {
		fmt.Printf("public key export\n")
	}
	fmt.Printf("fingerprint: %0X\n", fingerprint)
	fmt.Printf("path: %s\n", path)

	return nil
}

func exportRun(cmd *cobra.Command, args []string) {

	if err := doKeyExportCmd(secretExport, args[0], args[1]); err != nil {
		sylog.Errorf("key export command failed: %s", err)
		os.Exit(2)
	}

}

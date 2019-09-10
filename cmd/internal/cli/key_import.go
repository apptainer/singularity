// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/sypgp"
)

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

var keyImportWithNewPassword bool
var keyImportWithNewPasswordFlag = cmdline.Flag{
	ID:           "keyImportWithNewPasswordFlag",
	Value:        &keyImportWithNewPassword,
	DefaultValue: false,
	Name:         "new-password",
	Usage:        `set a new password to the private key`,
}

func importRun(cmd *cobra.Command, args []string) {
	keyring := sypgp.NewHandle("")
	if err := keyring.ImportKey(args[0], keyImportWithNewPassword); err != nil {
		sylog.Errorf("key import command failed: %s", err)
		os.Exit(2)
	}

}

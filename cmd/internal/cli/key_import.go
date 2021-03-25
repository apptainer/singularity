// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2019-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
)

// KeyImportCmd is `singularity key (or keys) import` and imports a local key into the singularity keyring.
var KeyImportCmd = &cobra.Command{
	PreRun:                checkGlobal,
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
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
	var opts []sypgp.HandleOpt
	path := ""

	if keyGlobalPubKey {
		path = buildcfg.SINGULARITY_CONFDIR
		opts = append(opts, sypgp.GlobalHandleOpt())
	}

	keyring := sypgp.NewHandle(path, opts...)
	if err := keyring.ImportKey(args[0], keyImportWithNewPassword); err != nil {
		sylog.Errorf("key import command failed: %s", err)
		os.Exit(2)
	}

}

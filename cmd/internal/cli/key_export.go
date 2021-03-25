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

var secretExport bool
var armor bool

// -s|--secret
var keyExportSecretFlag = cmdline.Flag{
	ID:           "keyExportSecretFlag",
	Value:        &secretExport,
	DefaultValue: false,
	Name:         "secret",
	ShortHand:    "s",
	Usage:        "export a secret key",
}

// -a|--armor
var keyExportArmorFlag = cmdline.Flag{
	ID:           "keyExportArmorFlag",
	Value:        &armor,
	DefaultValue: false,
	Name:         "armor",
	ShortHand:    "a",
	Usage:        "ascii armored format",
}

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterFlagForCmd(&keyExportSecretFlag, KeyExportCmd)
		cmdManager.RegisterFlagForCmd(&keyExportArmorFlag, KeyExportCmd)
	})
}

// KeyExportCmd is `singularity key export` and exports a public or secret
// key from local keyring.
var KeyExportCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run:                   exportRun,

	Use:     docs.KeyExportUse,
	Short:   docs.KeyExportShort,
	Long:    docs.KeyExportLong,
	Example: docs.KeyExportExample,
}

func exportRun(cmd *cobra.Command, args []string) {
	var opts []sypgp.HandleOpt
	path := ""

	if keyGlobalPubKey {
		path = buildcfg.SINGULARITY_CONFDIR
		opts = append(opts, sypgp.GlobalHandleOpt())
	}

	keyring := sypgp.NewHandle(path, opts...)
	if secretExport {
		err := keyring.ExportPrivateKey(args[0], armor)
		if err != nil {
			sylog.Errorf("key export command failed: %s", err)
			os.Exit(10)
		}
	} else {
		err := keyring.ExportPubKey(args[0], armor)
		if err != nil {
			sylog.Errorf("key export command failed: %s", err)
			os.Exit(10)
		}
	}
}

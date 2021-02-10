// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/sylog"
)

var (
	keyServerURI        string // -u command line option
	keySearchLongList   bool   // -l option for long-list
	keyNewpairBitLength int    // -b option for bit length
	keyGlobalPubKey     bool   // -g option to manage global public keys
)

// -u|--url
var keyServerURIFlag = cmdline.Flag{
	ID:           "keyServerURIFlag",
	Value:        &keyServerURI,
	DefaultValue: "",
	Name:         "url",
	ShortHand:    "u",
	Usage:        "specify the key server URL",
	EnvKeys:      []string{"URL"},
}

// -l|--long-list
var keySearchLongListFlag = cmdline.Flag{
	ID:           "keySearchLongListFlag",
	Value:        &keySearchLongList,
	DefaultValue: false,
	Name:         "long-list",
	ShortHand:    "l",
	Usage:        "output long list when searching for keys",
}

// -b|--bit-length
var keyNewpairBitLengthFlag = cmdline.Flag{
	ID:           "keyNewpairBitLengthFlag",
	Value:        &keyNewpairBitLength,
	DefaultValue: 4096,
	Name:         "bit-length",
	ShortHand:    "b",
	Usage:        "specify key bit length",
}

// -g|--global
var keyGlobalPubKeyFlag = cmdline.Flag{
	ID:           "keyGlobalPubKeyFlag",
	Value:        &keyGlobalPubKey,
	DefaultValue: false,
	Name:         "global",
	ShortHand:    "g",
	Usage:        "manage global public keys (import/pull/remove are restricted to root user or unprivileged installation only)",
}

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterCmd(KeyCmd)

		cmdManager.RegisterSubCmd(KeyCmd, KeyNewPairCmd)
		cmdManager.RegisterFlagForCmd(KeyNewPairNameFlag, KeyNewPairCmd)
		cmdManager.RegisterFlagForCmd(KeyNewPairEmailFlag, KeyNewPairCmd)
		cmdManager.RegisterFlagForCmd(KeyNewPairCommentFlag, KeyNewPairCmd)
		cmdManager.RegisterFlagForCmd(KeyNewPairPasswordFlag, KeyNewPairCmd)
		cmdManager.RegisterFlagForCmd(KeyNewPairPushFlag, KeyNewPairCmd)

		cmdManager.RegisterSubCmd(KeyCmd, KeyListCmd)
		cmdManager.RegisterSubCmd(KeyCmd, KeySearchCmd)
		cmdManager.RegisterSubCmd(KeyCmd, KeyPullCmd)
		cmdManager.RegisterSubCmd(KeyCmd, KeyPushCmd)
		cmdManager.RegisterSubCmd(KeyCmd, KeyImportCmd)
		cmdManager.RegisterSubCmd(KeyCmd, KeyRemoveCmd)
		cmdManager.RegisterSubCmd(KeyCmd, KeyExportCmd)

		cmdManager.RegisterFlagForCmd(&keyServerURIFlag, KeySearchCmd, KeyPushCmd, KeyPullCmd)
		cmdManager.RegisterFlagForCmd(&keySearchLongListFlag, KeySearchCmd)
		cmdManager.RegisterFlagForCmd(&keyNewpairBitLengthFlag, KeyNewPairCmd)
		cmdManager.RegisterFlagForCmd(&keyImportWithNewPasswordFlag, KeyImportCmd)

		cmdManager.RegisterFlagForCmd(
			&keyGlobalPubKeyFlag,
			KeyImportCmd, KeyExportCmd, KeyListCmd, KeyPullCmd, KeyPushCmd, KeyRemoveCmd,
		)
	})
}

func checkGlobal(cmd *cobra.Command, args []string) {
	if !keyGlobalPubKey || os.Geteuid() == 0 || buildcfg.SINGULARITY_SUID_INSTALL == 0 {
		return
	}
	path := cmd.CommandPath()
	sylog.Fatalf("%q command with --global requires root privileges or an unprivileged installation", path)
}

// KeyCmd is the 'key' command that allows management of keyrings
var KeyCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Invalid command")
	},
	DisableFlagsInUseLine: true,
	Aliases:               []string{"keys"},

	Use:           docs.KeyUse,
	Short:         docs.KeyShort,
	Long:          fmt.Sprintf(docs.KeyLong, buildcfg.SYSCONFDIR),
	Example:       docs.KeyExample,
	SilenceErrors: true,
}

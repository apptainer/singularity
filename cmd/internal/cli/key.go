// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/pkg/cmdline"
)

const (
	defaultKeyServer = "https://keys.sylabs.io"
)

var (
	keyServerURI        string // -u command line option
	keySearchLongList   bool   // -l option for long-list
	keyNewpairBitLength int    // -b option for bit length
)

// -u|--url
var keyServerURIFlag = cmdline.Flag{
	ID:           "keyServerURIFlag",
	Value:        &keyServerURI,
	DefaultValue: defaultKeyServer,
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

func init() {
	cmdManager.RegisterCmd(KeyCmd)
	cmdManager.RegisterSubCmd(KeyCmd, KeyNewPairCmd)
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
}

// KeyCmd is the 'key' command that allows management of key stores
var KeyCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Invalid command")
	},
	DisableFlagsInUseLine: true,
	Aliases:               []string{"keys"},

	Use:           docs.KeyUse,
	Short:         docs.KeyShort,
	Long:          docs.KeyLong,
	Example:       docs.KeyExample,
	SilenceErrors: true,
}

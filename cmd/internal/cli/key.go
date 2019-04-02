// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
)

const (
	defaultKeyServer = "https://keys.sylabs.io"
)

var (
	keyServerURI string // -u command line option
)

func init() {
	SingularityCmd.AddCommand(KeyCmd)

	// key commands
	KeyCmd.AddCommand(KeyNewPairCmd)
	KeyCmd.AddCommand(KeyListCmd)
	KeyCmd.AddCommand(KeySearchCmd)
	KeyCmd.AddCommand(KeyPullCmd)
	KeyCmd.AddCommand(KeyPushCmd)
	KeyCmd.AddCommand(KeyImportCmd)
	KeyCmd.AddCommand(KeyExportCmd)
	KeyCmd.AddCommand(KeyRemoveCmd)
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

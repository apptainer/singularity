// Copyright (c) 2018, Sylabs Inc. All rights reserved.
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
	defaultKeysServer = "https://keys.sylabs.io"
)

var (
	keyServerURL string // -u command line option
)

func init() {
	SingularityCmd.AddCommand(KeysCmd)
	SingularityCmd.AddCommand(KeyCmd)
	KeysCmd.AddCommand(KeysNewPairCmd)
	KeysCmd.AddCommand(KeysListCmd)
	KeysCmd.AddCommand(KeysSearchCmd)
	KeysCmd.AddCommand(KeysPullCmd)
	KeysCmd.AddCommand(KeysPushCmd)
	KeyCmd.AddCommand(KeysNewPairCmd)
	KeyCmd.AddCommand(KeysListCmd)
	KeyCmd.AddCommand(KeysSearchCmd)
	KeyCmd.AddCommand(KeysPullCmd)
	KeyCmd.AddCommand(KeysPushCmd)
}

// KeysCmd is the 'keys' command that allows management of key stores
var KeysCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Invalid command")
	},
	DisableFlagsInUseLine: true,
	Hidden: true,

	Use:           docs.KeysUse,
	Short:         docs.KeyShort,
	Long:          docs.KeyLong,
	Example:       docs.KeyExample,
	SilenceErrors: true,
}

var KeyCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Invalid command")
	},
	DisableFlagsInUseLine: true,

	Use:           docs.KeyUse,
	Short:         docs.KeyShort,
	Long:          docs.KeyLong,
	Example:       docs.KeyExample,
	SilenceErrors: true,
}

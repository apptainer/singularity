// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/singularityware/singularity/src/docs"
	"github.com/spf13/cobra"
)

const (
	defaultKeysServer = "https://keys.sylabs.io"
)

var (
	keyServerURL string // -u command line option
)

func init() {
	SingularityCmd.AddCommand(KeysCmd)
	KeysCmd.AddCommand(KeysNewPairCmd)
	KeysCmd.AddCommand(KeysListCmd)
	KeysCmd.AddCommand(KeysSearchCmd)
	KeysCmd.AddCommand(KeysPullCmd)
	KeysCmd.AddCommand(KeysPushCmd)
}

// KeysCmd is the 'keys' command that allows management of key stores
var KeysCmd = &cobra.Command{
	Run: nil,
	DisableFlagsInUseLine: true,

	Use:     docs.KeysUse,
	Short:   docs.KeysShort,
	Long:    docs.KeysLong,
	Example: docs.KeysExample,
}

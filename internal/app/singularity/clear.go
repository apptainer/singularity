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

//const (
//	defaultKeysServer = "https://keys.sylabs.io"
//)

//var (
//	keyServerURL string // -u command line option
//)

func init() {
	SingularityCmd.AddCommand(ClearCmd)
	ClearCmd.AddCommand(ClearCacheCmd)
}

// ClearCmd is the 'clear' command that will clear your local singularity cache
var ClearCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Invalid command")
	},
	DisableFlagsInUseLine: true,

	Use:           docs.ClearUse,
	Short:         docs.ClearShort,
	Long:          docs.ClearLong,
	Example:       docs.ClearExample,
	SilenceErrors: true,
}


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

func init() {
	SingularityCmd.AddCommand(CacheCmd)
	CacheCmd.AddCommand(CacheCleanCmd)
	CacheCmd.AddCommand(CacheListCmd)
}

var CacheCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Invalid command")
	},
	DisableFlagsInUseLine: true,

	Use:           docs.CacheUse,
	Short:         docs.CacheShort,
	Long:          docs.CacheLong,
	Example:       docs.CacheExample,
	SilenceErrors: true,
}



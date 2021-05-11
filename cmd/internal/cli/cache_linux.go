// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"errors"

	"github.com/hpcng/singularity/docs"
	"github.com/hpcng/singularity/pkg/cmdline"
	"github.com/spf13/cobra"
)

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterCmd(CacheCmd)
		cmdManager.RegisterSubCmd(CacheCmd, cacheCleanCmd)
		cmdManager.RegisterSubCmd(CacheCmd, CacheListCmd)
	})
}

// CacheCmd : aka, `singularity cache`
var CacheCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("invalid command")
	},
	DisableFlagsInUseLine: true,

	Use:           docs.CacheUse,
	Short:         docs.CacheShort,
	Long:          docs.CacheLong,
	Example:       docs.CacheExample,
	SilenceErrors: true,
}

// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/cleanCache"
)

var singularityCache bool

func init() {
	ClearCacheCmd.Flags().SetInterspersed(false)
	ClearCacheCmd.Flags().BoolVarP(&singularityCache, "cache", "c", false, "clear clcche")
//	ClearCacheCmd.Flags().SetAnnotation("singularityCache", "envfoofoo", []string{"CACHE"})
}

// ClearCacheCmd is `singularity clear cache' and will clear your local singularity cache
var ClearCacheCmd = &cobra.Command {
	Args:                  cobra.ExactArgs(0),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := clearCacheCmd(singularityCache); err != nil {
			os.Exit(2)
		}
	},

	Use:     docs.ClearCacheUse,
	Short:   docs.ClearCacheShort,
	Long:    docs.ClearCacheLong,
	Example: docs.ClearCacheExample,
}

func clearCacheCmd(singularityCache bool) error {
	err := cleanCache.CleanSingularityCache()
	if err != nil {
	    sylog.Fatalf("%v", err)
	    os.Exit(255)
	}
	return nil
}


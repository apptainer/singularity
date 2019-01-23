// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/cacheCli"
)

var (
	allClean bool
	libraryClean bool
	ociClean bool
)

func init() {
	CacheCleanCmd.Flags().SetInterspersed(false)

	CacheCleanCmd.Flags().BoolVarP(&allClean, "all", "a", false, "clean all cache (default)")
	CacheCleanCmd.Flags().SetAnnotation("library", "envkey", []string{"LIBRARY"})

	CacheCleanCmd.Flags().BoolVarP(&libraryClean, "library", "l", false, "clean library cache")
	CacheCleanCmd.Flags().SetAnnotation("library", "envkey", []string{"LIBRARY"})

	CacheCleanCmd.Flags().BoolVarP(&ociClean, "oci", "d", false, "clean oci/docker cache")
	CacheCleanCmd.Flags().SetAnnotation("oci", "envkey", []string{"OCI"})
}

// ClearCacheCmd is `singularity cache clean' and will clear your local singularity cache
var CacheCleanCmd = &cobra.Command {
	Args:                  cobra.ExactArgs(0),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cacheCleanCmd(); err != nil {
			os.Exit(2)
		}
	},

	Use:     docs.CacheCleanUse,
	Short:   docs.CacheCleanShort,
	Long:    docs.CacheCleanLong,
	Example: docs.CacheCleanExample,
}


//func cacheCleanCmd(cacheClean bool) error {
func cacheCleanCmd() error {
	err := cacheCli.CleanSingularityCache(allClean, libraryClean, ociClean)
	if err != nil {
		sylog.Fatalf("%v", err)
		os.Exit(255)
	}

	return err
}


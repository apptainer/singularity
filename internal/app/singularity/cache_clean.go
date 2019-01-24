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
	typeNameClean string
//	libraryClean bool
//	ociClean bool
	cacheName string
)

func init() {
	CacheCleanCmd.Flags().SetInterspersed(false)

	CacheCleanCmd.Flags().BoolVarP(&allClean, "all", "a", false, "clean all cache (not compatible with any other flags)")
	CacheCleanCmd.Flags().SetAnnotation("all", "envkey", []string{"ALL"})

//	CacheCleanCmd.Flags().BoolVarP(&libraryClean, "library", "l", false, "only clean cache from library")
//	CacheCleanCmd.Flags().SetAnnotation("library", "envkey", []string{"LIBRARY"})

//	CacheCleanCmd.Flags().BoolVarP(&ociClean, "oci", "d", false, "only clean cache from docker/oci")
//	CacheCleanCmd.Flags().SetAnnotation("oci", "envkey", []string{"OCI"})

	CacheCleanCmd.Flags().StringVarP(&typeNameClean, "type", "t", "", "specify a cache type, choose between: library, and oci")
	CacheCleanCmd.Flags().SetAnnotation("type", "envkey", []string{"TYPE"})

	CacheCleanCmd.Flags().StringVarP(&cacheName, "name", "n", "", "specify a container cache to clean (will clear all cache with the same name)")
	CacheCleanCmd.Flags().SetAnnotation("name", "envkey", []string{"NAME"})

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


func cacheCleanCmd() error {

//	err := cacheCli.CleanSingularityCache(allClean, libraryClean, ociClean, cacheName)
	err := cacheCli.CleanSingularityCache(allClean, typeNameClean, cacheName)
	if err != nil {
		sylog.Fatalf("%v", err)
		os.Exit(255)
	}

	return err
}


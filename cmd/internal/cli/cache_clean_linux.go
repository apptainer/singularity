// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"

	"github.com/sylabs/singularity/pkg/cmdline"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

var (
	cleanAll        bool
	cacheCleanTypes []string
	cacheName       []string
)

// -a|--all
var cacheCleanAllFlag = cmdline.Flag{
	ID:           "cacheCleanAllFlag",
	Value:        &cleanAll,
	DefaultValue: false,
	Name:         "all",
	ShortHand:    "a",
	Usage:        "clean all cache (will override all other options)",
	EnvKeys:      []string{"ALL"},
}

// -T|--type
var cacheCleanTypesFlag = cmdline.Flag{
	ID:           "cacheCleanTypesFlag",
	Value:        &cacheCleanTypes,
	DefaultValue: []string{"blob"},
	Name:         "type",
	ShortHand:    "T",
	Usage:        "clean cache type, choose between: library, oci, shub, and blob",
	EnvKeys:      []string{"TYPE"},
}

// -N|--name
var cacheCleanNameFlag = cmdline.Flag{
	ID:           "cacheCleanNameFlag",
	Value:        &cacheName,
	DefaultValue: []string{},
	Name:         "name",
	ShortHand:    "N",
	Usage:        "specify a container cache to clean (will clear all cache with the same name)",
	EnvKeys:      []string{"NAME"},
}

func init() {
	cmdManager.RegisterFlagForCmd(&cacheCleanAllFlag, CacheCleanCmd)
	cmdManager.RegisterFlagForCmd(&cacheCleanTypesFlag, CacheCleanCmd)
	cmdManager.RegisterFlagForCmd(&cacheCleanNameFlag, CacheCleanCmd)
}

// CacheCleanCmd is 'singularity cache clean' and will clear your local singularity cache
var CacheCleanCmd = &cobra.Command{
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
	// We create a handle to access the current image cache
	imgCache, err := cache.HdlInit(os.Getenv(cache.DirEnv))
	if imgCache == nil || err != nil {
		sylog.Fatalf("failed to create an image cache handle")
	}
	err = singularity.CleanSingularityCache(imgCache, cleanAll, cacheCleanTypes, cacheName)
	if err != nil {
		sylog.Fatalf("Failed while clean cache: %v", err)
	}

	return err
}

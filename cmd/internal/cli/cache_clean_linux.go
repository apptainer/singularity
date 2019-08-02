// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/cmdline"
)

var (
	cacheCleanDry   bool
	cacheCleanTypes []string
	cacheCleanNames []string
)

// -T|--type
var cacheCleanTypesFlag = cmdline.Flag{
	ID:           "cacheCleanTypes",
	Value:        &cacheCleanTypes,
	DefaultValue: []string{"all"},
	Name:         "type",
	ShortHand:    "T",
	Usage:        "a list of cache types to clean (possible values: library, oci, shub, blob, net, oras, all)",
}

// -N|--name
var cacheCleanNameFlag = cmdline.Flag{
	ID:           "cacheCleanNameFlag",
	Value:        &cacheCleanNames,
	DefaultValue: []string{},
	Name:         "name",
	ShortHand:    "N",
	Usage:        "specify a container cache to clean (will clear all cache with the same name)",
}

// --dry-run
var cacheCleanDryFlag = cmdline.Flag{
	ID:           "cacheCleanDryFlag",
	Value:        &cacheCleanDry,
	DefaultValue: false,
	Name:         "dry-run",
	Usage:        "operate in dry run mode and do not actually clean the cache",
}

func init() {
	cmdManager.RegisterFlagForCmd(&cacheCleanTypesFlag, CacheCleanCmd)
	cmdManager.RegisterFlagForCmd(&cacheCleanNameFlag, CacheCleanCmd)
	cmdManager.RegisterFlagForCmd(&cacheCleanDryFlag, CacheCleanCmd)
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
	imgCache := getCacheHandle()
	err := singularity.CleanSingularityCache(imgCache, !cacheCleanDry, cacheCleanTypes, cacheCleanNames)
	if err != nil {
		sylog.Fatalf("Failed while clean cache: %v", err)
	}
	return nil
}

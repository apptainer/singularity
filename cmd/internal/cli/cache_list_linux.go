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
	cacheListTypes   []string
	cacheListVerbose bool
)

// -T|--type
var cacheListTypesFlag = cmdline.Flag{
	ID:           "cacheListTypes",
	Value:        &cacheListTypes,
	DefaultValue: []string{"all"},
	Name:         "type",
	ShortHand:    "T",
	Usage:        "a list of cache types to display, possible entries: library, oci, shub, blob(s), all",
}

// -s|--summary
var cacheListVerboseFlag = cmdline.Flag{
	ID:           "cacheListVerbose",
	Value:        &cacheListVerbose,
	DefaultValue: false,
	Name:         "verbose",
	ShortHand:    "v",
	Usage:        "include cache entries in the output",
}

func init() {
	cmdManager.RegisterFlagForCmd(&cacheListTypesFlag, CacheListCmd)
	cmdManager.RegisterFlagForCmd(&cacheListVerboseFlag, CacheListCmd)
}

// CacheListCmd is 'singularity cache list' and will list your local singularity cache
var CacheListCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cacheListCmd(); err != nil {
			os.Exit(2)
		}
	},

	Use:     docs.CacheListUse,
	Short:   docs.CacheListShort,
	Long:    docs.CacheListLong,
	Example: docs.CacheListExample,
}

func cacheListCmd() error {
	// A get a handle for the current image cache
	imgCache := getCacheHandle()
	if imgCache == nil {
		sylog.Fatalf("failed to create image cache handle")
	}

	err := singularity.ListSingularityCache(imgCache, cacheListTypes, cacheListVerbose)
	if err != nil {
		sylog.Fatalf("An error occurred while listing cache: %v", err)
		return err
	}
	return nil
}

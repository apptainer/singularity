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
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/cmdline"
)

var (
	cacheListTypes   []string
	allList          bool
	cacheListSummary bool
)

// -T|--type
var cacheListTypesFlag = cmdline.Flag{
	ID:           "cacheListTypes",
	Value:        &cacheListTypes,
	DefaultValue: []string{"library", "oci", "shub", "blobSum"},
	Name:         "type",
	ShortHand:    "T",
	Usage:        "a list of cache types to display, possible entries: library, oci, shub, blob(s), blobSum, all",
	EnvKeys:      []string{"TYPE"},
}

// -s|--summary
var cacheListSummaryFlag = cmdline.Flag{
	ID:           "cacheListSummary",
	Value:        &cacheListSummary,
	DefaultValue: false,
	Name:         "summary",
	ShortHand:    "s",
	Usage:        "display a cache summary",
}

// -a|--all
var cacheListAllFlag = cmdline.Flag{
	ID:           "cacheListAllFlag",
	Value:        &allList,
	DefaultValue: false,
	Name:         "all",
	ShortHand:    "a",
	Usage:        "list all cache types",
	EnvKeys:      []string{"ALL"},
}

func init() {
	cmdManager.RegisterFlagForCmd(&cacheListTypesFlag, CacheListCmd)
	cmdManager.RegisterFlagForCmd(&cacheListSummaryFlag, CacheListCmd)
	cmdManager.RegisterFlagForCmd(&cacheListAllFlag, CacheListCmd)
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
	imgCache, err := cache.HdlInit(os.Getenv(cache.DirEnv))
	if imgCache == nil || err != nil {
		sylog.Fatalf("failed to create image cache handle")
	}

	err = singularity.ListSingularityCache(imgCache, cacheListTypes, allList, cacheListSummary)
	if err != nil {
		sylog.Fatalf("An error occurred while listing cache: %v", err)
		return err
	}
	return nil
}

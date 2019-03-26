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
)

var (
	cacheListTypes   []string
	allList          bool
	cacheListSummary bool
)

func init() {
	CacheListCmd.Flags().SetInterspersed(false)

	CacheListCmd.Flags().StringSliceVarP(&cacheListTypes, "type", "T", []string{"library", "oci", "blobSum"},
		"a list of cache types to display, possible entries: library, oci, blob(s), blobSum, all")
	CacheListCmd.Flags().SetAnnotation("type", "envkey", []string{"TYPE"})
	CacheListCmd.Flags().BoolVarP(&cacheListSummary, "summary", "s", false, "display a cache summary")

	CacheListCmd.Flags().BoolVarP(&allList, "all", "a", false, "list all cache types")
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
	err := singularity.ListSingularityCache(cacheListTypes, allList, cacheListSummary)
	if err != nil {
		sylog.Fatalf("Not listing cache; an error occurred: %v", err)
		return err
	}
	return err
}

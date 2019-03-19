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

	CacheListCmd.Flags().StringSliceVarP(&cacheListTypes, "type", "T", []string{"library", "oci"}, "list of cache types, choose between: library, oci, and blob. Multiple values can be specified separating them with a comma.")
	CacheListCmd.Flags().SetAnnotation("type", "envkey", []string{"CACHE_LIST_TYPE"})

	CacheListCmd.Flags().BoolVarP(&cacheListSummary, "summery", "s", false, "display a cache summary")

	CacheListCmd.Flags().BoolVarP(&allList, "all", "a", false, "list all cache types")
}

// CacheListCmd is 'singularity cache list' and will list your local singularity cache
var CacheListCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(0),
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

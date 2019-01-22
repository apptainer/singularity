// Copyright (c) 2016???-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
)

// ClearListCmd is `singularity cache list' and will list your local singularity cache
var CacheListCmd = &cobra.Command {
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

	sylog.Infof("HELLO WORLD FROM CACHE LIST!!!!")

//	sylog.Infof("OciBlob(): %v", cache.OciBlob())

//	err := cleanCache.CleanSingularityCache()
//	if err != nil {
//	    sylog.Fatalf("%v", err)
//	    os.Exit(255)
//	}
	return nil
}




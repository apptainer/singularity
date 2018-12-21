// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/client/library"
	"github.com/sylabs/singularity/docs"
)

var (
	// SearchLibraryURI holds the base URI to a Sylabs library API instance
	SearchLibraryURI string
)

func init() {
	SearchCmd.Flags().SetInterspersed(false)

	SearchCmd.Flags().StringVar(&SearchLibraryURI, "library", "https://library.sylabs.io", "URI for library to search")
	SearchCmd.Flags().SetAnnotation("library", "envkey", []string{"LIBRARY"})

	SingularityCmd.AddCommand(SearchCmd)
}

// SearchCmd singularity search
var SearchCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		if err := client.SearchLibrary(args[0], SearchLibraryURI, authToken); err != nil {
			sylog.Fatalf("Couldn't search library: %v", err)
		}

	},

	Use:     docs.SearchUse,
	Short:   docs.SearchShort,
	Long:    docs.SearchLong,
	Example: docs.SearchExample,
}

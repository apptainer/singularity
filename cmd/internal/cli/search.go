// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	client "github.com/sylabs/singularity/pkg/client/library"
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
		handleSearchFlags(cmd)

		if err := client.SearchLibrary(args[0], SearchLibraryURI, authToken); err != nil {
			sylog.Fatalf("Couldn't search library: %v", err)
		}

	},

	Use:     docs.SearchUse,
	Short:   docs.SearchShort,
	Long:    docs.SearchLong,
	Example: docs.SearchExample,
}

func handleSearchFlags(cmd *cobra.Command) {
	// if we can load config and if default endpoint is set, use that
	// otherwise fall back on regular authtoken and URI behavior
	endpoint, err := sylabsRemote(remoteConfig)
	if err == scs.ErrNoDefault {
		sylog.Warningf("No default remote in use, falling back to: %v", SearchLibraryURI)
		return
	} else if err != nil {
		sylog.Fatalf("Unable to load remote configuration: %v", err)
	}

	authToken = endpoint.Token
	if !cmd.Flags().Lookup("library").Changed {
		uri, err := endpoint.GetServiceURI("library")
		if err != nil {
			sylog.Fatalf("Unable to get library URI: %v", err)
		}
		SearchLibraryURI = uri
	}
}

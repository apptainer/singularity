// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
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
	// PushLibraryURI holds the base URI to a Sylabs library API instance
	PushLibraryURI string
)

func init() {
	PushCmd.Flags().SetInterspersed(false)

	PushCmd.Flags().StringVar(&PushLibraryURI, "library", "https://library.sylabs.io", "the library to push to")
	PushCmd.Flags().SetAnnotation("library", "envkey", []string{"LIBRARY"})

	SingularityCmd.AddCommand(PushCmd)
}

// PushCmd singularity push
var PushCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(2),
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {

		// if we can load config and if default endpoint is set, use that
		// otherwise fall back on regular authtoken and URI behavior
		e, err := sylabsRemote(remoteConfig)
		if err == nil {
			authToken = e.Token
			if !cmd.Flags().Lookup("library").Changed {
				uri, err := e.GetServiceURI("library")
				if err != nil {
					sylog.Fatalf("Unable to get library service URI: %v", err)
				}
				PushLibraryURI = uri
			}
		} else if err == scs.ErrNoDefault {
			sylog.Warningf("No default remote in use, falling back to: %v", PushLibraryURI)
		} else {
			sylog.Debugf("Unable to load remote configuration: %v", err)
		}

		// Push to library requires a valid authToken
		if authToken != "" {
			err := client.UploadImage(args[0], args[1], PushLibraryURI, authToken, "No Description")
			if err != nil {
				sylog.Fatalf("%v\n", err)
			}
		} else {
			sylog.Fatalf("Couldn't push image to library: %v", authWarning)
		}
	},

	Use:     docs.PushUse,
	Short:   docs.PushShort,
	Long:    docs.PushLong,
	Example: docs.PushExample,
}

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
	"github.com/sylabs/singularity/pkg/cmdline"
)

var (
	// PushLibraryURI holds the base URI to a Sylabs library API instance
	PushLibraryURI string
)

// --library
var pushLibraryURIFlag = cmdline.Flag{
	ID:           "pushLibraryURIFlag",
	Value:        &PushLibraryURI,
	DefaultValue: "https://library.sylabs.io",
	Name:         "library",
	Usage:        "the library to push to",
	EnvKeys:      []string{"LIBRARY"},
}

func init() {
	cmdManager.RegisterCmd(PushCmd, false)

	flagManager.RegisterCmdFlag(&pushLibraryURIFlag, PushCmd)
}

// PushCmd singularity push
var PushCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(2),
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		handlePushFlags(cmd)

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

func handlePushFlags(cmd *cobra.Command) {
	// if we can load config and if default endpoint is set, use that
	// otherwise fall back on regular authtoken and URI behavior
	endpoint, err := sylabsRemote(remoteConfig)
	if err == scs.ErrNoDefault {
		sylog.Warningf("No default remote in use, falling back to: %v", PushLibraryURI)
		return
	} else if err != nil {
		sylog.Fatalf("Unable to load remote configuration: %v", err)
	}

	authToken = endpoint.Token
	if !cmd.Flags().Lookup("library").Changed {
		uri, err := endpoint.GetServiceURI("library")
		if err != nil {
			sylog.Fatalf("Unable to get library service URI: %v", err)
		}
		PushLibraryURI = uri
	}
}

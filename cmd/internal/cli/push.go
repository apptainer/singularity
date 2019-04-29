// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	client "github.com/sylabs/singularity/pkg/client/library"
	"github.com/sylabs/singularity/pkg/signing"
)

var (
	// PushLibraryURI holds the base URI to a Sylabs library API instance
	PushLibraryURI string

	// unauthenticatedPush when true; will never ask to push a unsigned container
	unauthenticatedPush bool
)

func init() {
	PushCmd.Flags().SetInterspersed(false)

	PushCmd.Flags().StringVar(&PushLibraryURI, "library", "https://library.sylabs.io", "the library to push to")
	PushCmd.Flags().SetAnnotation("library", "envkey", []string{"LIBRARY"})

	PushCmd.Flags().BoolVarP(&unauthenticatedPush, "allow-unsigned", "U", false, "do not require a signed container")
	PushCmd.Flags().SetAnnotation("allow-unsigned", "envkey", []string{"ALLOW_UNSIGNED"})

	SingularityCmd.AddCommand(PushCmd)
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
			if _, err := os.Stat(args[0]); os.IsNotExist(err) {
				sylog.Fatalf("Unable to open: %v: %v", args[0], err)
			}
			if !unauthenticatedPush {
				// check if the container is signed
				imageSigned, err := signing.IsSigned(args[0], KeyServerURL, 0, false, authToken, false, true)
				if err != nil {
					// err will be: "unable to verify container: %v", err
					sylog.Warningf("%v", err)
				}
				// if its not signed, print a warning
				if !imageSigned {
					sylog.Infof("TIP: Learn how to sign your own containers here : https://www.sylabs.io/docs/")
					fmt.Fprintf(os.Stderr, "\nUnable to verify your container! You REALLY should sign your container before pushing!\n")
					fmt.Fprintf(os.Stderr, "Stopping upload.\n")
					os.Exit(3)
				}
			} else {
				sylog.Warningf("Skipping container verifying")
			}

			err := client.UploadImage(args[0], args[1], PushLibraryURI, authToken, "No Description")
			if err != nil {
				sylog.Fatalf("%v\n", err)
			}
		} else {
			sylog.Fatalf("Couldn't push image to library: %v", remoteWarning)
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
		sylog.Debugf("using default key server url: %v", KeyServerURL)
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

	uri, err := endpoint.GetServiceURI("keystore")
	if err != nil {
		sylog.Warningf("Unable to get library service URI: %v, defaulting to %s.", err, KeyServerURL)
		return
	}
	KeyServerURL = uri
}

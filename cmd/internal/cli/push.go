// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/client/oras"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/sylabs/singularity/pkg/util/singularityconf"
)

var (
	// PushLibraryURI holds the base URI to a Sylabs library API instance
	PushLibraryURI string

	// unauthenticatedPush when true; will never ask to push a unsigned container
	unauthenticatedPush bool
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

// -U|--allow-unsigned
var pushAllowUnsignedFlag = cmdline.Flag{
	ID:           "pushAllowUnsignedFlag",
	Value:        &unauthenticatedPush,
	DefaultValue: false,
	Name:         "allow-unsigned",
	ShortHand:    "U",
	Usage:        "do not require a signed container",
	EnvKeys:      []string{"ALLOW_UNSIGNED"},
}

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterCmd(PushCmd)

		cmdManager.RegisterFlagForCmd(&pushLibraryURIFlag, PushCmd)
		cmdManager.RegisterFlagForCmd(&pushAllowUnsignedFlag, PushCmd)

		cmdManager.RegisterFlagForCmd(&dockerUsernameFlag, PushCmd)
		cmdManager.RegisterFlagForCmd(&dockerPasswordFlag, PushCmd)
	})
}

// PushCmd singularity push
var PushCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(2),
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.TODO()

		file, dest := args[0], args[1]

		transport, ref := uri.Split(dest)
		if transport == "" {
			sylog.Fatalf("bad uri %s", dest)
		}

		switch transport {
		case LibraryProtocol, "": // Handle pushing to a library
			handlePushFlags(cmd)

			// set the default keyserver URL if any
			config := singularityconf.GetCurrentConfig()

			if config != nil && config.DefaultKeyserver != "" {
				keyServerURL = strings.TrimSuffix(config.DefaultKeyserver, "/")
			}

			clients, err := getKeyServerClients(keyServerURL, false)
			if err != nil {
				sylog.Fatalf("%s", err)
			}
			vopt := singularity.OptVerifyUseKeyServers(clients)

			err = singularity.LibraryPush(ctx, file, dest, authToken, PushLibraryURI, remoteWarning, unauthenticatedPush, vopt)
			if err == singularity.ErrLibraryUnsigned {
				fmt.Printf("TIP: You can push unsigned images with 'singularity push -U %s'.\n", file)
				fmt.Printf("TIP: Learn how to sign your own containers by using 'singularity help sign'\n\n")
				sylog.Fatalf("Unable to upload container: unable to verify signature")
				os.Exit(3)
			} else if err != nil {
				sylog.Fatalf("Unable to push image to library: %v", err)
			}
		case OrasProtocol:
			ociAuth, err := makeDockerCredentials(cmd)
			if err != nil {
				sylog.Fatalf("Unable to make docker oci credentials: %s", err)
			}

			if err := oras.UploadImage(file, ref, ociAuth); err != nil {
				sylog.Fatalf("Unable to push image to oci registry: %v", err)
			}
			sylog.Infof("Upload complete")
		default:
			sylog.Fatalf("Unsupported transport type: %s", transport)
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

	if !cmd.Flags().Lookup("library").Changed {
		uri, err := endpoint.GetServiceURI("library")
		if err != nil {
			sylog.Fatalf("Unable to get library service URI: %v", err)
		}
		PushLibraryURI = uri
		authToken = endpoint.Token
	}
}

// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	client "github.com/sylabs/singularity/pkg/client/library"
	"github.com/sylabs/singularity/pkg/signing"
)

const (
	// SifDefaultTag is the tag to use when a tag is not specified
	SifDefaultTag = "latest"

	// SifConfigMediaType is the config descriptor mediaType
	SifConfigMediaType = "application/vnd.sylabs.sif.config.v1+json"

	// SifLayerMediaType is the mediaType for the "layer" which contains the actual SIF file
	SifLayerMediaType = "appliciation/vnd.sylabs.sif.layer.tar+gzip"
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

	PushCmd.Flags().AddFlag(actionFlags.Lookup("docker-username"))
	PushCmd.Flags().AddFlag(actionFlags.Lookup("docker-password"))
	PushCmd.Flags().AddFlag(actionFlags.Lookup("docker-login"))

	SingularityCmd.AddCommand(PushCmd)
}

// PushCmd singularity push
var PushCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(2),
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		file, dest := args[0], args[1]

		transport, ref := uri.Split(dest)
		if transport == "" {
			sylog.Fatalf("bad uri %s", dest)
		}

		switch transport {
		case LibraryProtocol, "": // Handle pushing to a library
			handlePushFlags(cmd)

			// Push to library requires a valid authToken
			if authToken == "" {
				sylog.Fatalf("Couldn't push image to library: %v", remoteWarning)
			}

			if _, err := os.Stat(file); os.IsNotExist(err) {
				sylog.Fatalf("Unable to open: %v: %v", file, err)
			}

			if !unauthenticatedPush {
				// check if the container is signed
				imageSigned, err := signing.IsSigned(file, KeyServerURL, 0, false, authToken, true)
				if err != nil {
					// err will be: "unable to verify container: %v", err
					sylog.Warningf("%v", err)
				}

				// if its not signed, print a warning
				if !imageSigned {
					sylog.Infof("TIP: Learn how to sign your own containers here : https://www.sylabs.io/docs/")
					fmt.Fprintf(os.Stderr, "\nUnable to verify Your container! You REALLY should sign your container before pushing!\n")
					fmt.Fprintf(os.Stderr, "Stoping upload.\n")
					os.Exit(3)
				}
			} else {
				sylog.Warningf("Skipping container verifying")
			}

			if err := client.UploadImage(file, dest, PushLibraryURI, authToken, "No Description"); err != nil {
				sylog.Fatalf("%v\n", err)
			}

			return
		case OrasProtocol:
			ref = strings.TrimPrefix(ref, "//")

			ociAuth, err := makeDockerCredentials(cmd)
			if err != nil {
				sylog.Fatalf("Unable to make docker oci credentials: %s", err)
			}

			credFn := func(_ string) (string, string, error) {
				return ociAuth.Username, ociAuth.Password, nil
			}

			resolver := docker.NewResolver(docker.ResolverOptions{Credentials: credFn})

			store := content.NewFileStore("")
			defer store.Close()

			conf, err := store.Add("$config", SifConfigMediaType, "/dev/null")
			conf.Annotations = nil

			desc, err := store.Add(file, SifLayerMediaType, file)
			if err != nil {
				sylog.Fatalf("Unable to add file to store: %s", err)
			}

			descriptors := []ocispec.Descriptor{desc}

			if _, err := oras.Push(context.Background(), resolver, ref, store, descriptors, oras.WithConfig(conf)); err != nil {
				sylog.Fatalf("Unable to push: %s", err)
			}
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

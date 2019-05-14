// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/containerd/containerd/reference"
	"github.com/containerd/containerd/remotes/docker"
	ocitypes "github.com/containers/image/types"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/context"
	"github.com/deislabs/oras/pkg/oras"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	client "github.com/sylabs/singularity/pkg/client/library"
	"github.com/sylabs/singularity/pkg/signing"
)

const (
	// SifDefaultTag is the tag to use when a tag is not specified
	SifDefaultTag = "latest"

	// SifConfigMediaType is the config descriptor mediaType
	SifConfigMediaType = "application/vnd.sylabs.sif.config.v1+json"

	// SifLayerMediaType is the mediaType for the "layer" which contains the actual SIF file
	SifLayerMediaType = "appliciation/vnd.sylabs.sif.layer.tar"
)

var (
	// ErrLibraryUnsigned indicated that the image intended to be used is
	// not signed, nor has an override for requiring a signature been provided
	ErrLibraryUnsigned = errors.New("image is not signed")
)

// LibraryPush will upload the image specified by file to the library specified by libraryURI.
// Before uploading, the image will be checked for a valid signature, unless specified not to by the
// unauthenticated bool
func LibraryPush(file, dest, authToken, libraryURI, keyServerURL, remoteWarning string, unauthenticated bool) error {
	// Push to library requires a valid authToken
	if authToken == "" {
		return fmt.Errorf("couldn't push image to library: %v", remoteWarning)
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return fmt.Errorf("unable to open: %v: %v", file, err)
	}

	if !unauthenticated {
		// check if the container is signed
		imageSigned, err := signing.IsSigned(file, keyServerURL, 0, false, authToken, true)
		if err != nil {
			// err will be: "unable to verify container: %v", err
			sylog.Warningf("%v", err)
		}

		// if its not signed, print a warning
		if !imageSigned {
			sylog.Infof("TIP: Learn how to sign your own containers here : https://www.sylabs.io/docs/")
			return ErrLibraryUnsigned
		}
	} else {
		sylog.Warningf("Skipping container verifying")
	}

	if err := client.UploadImage(file, dest, libraryURI, authToken, "No Description"); err != nil {
		return err
	}

	return nil
}

// OrasPush upload the image specified by file and push it to the provided oci reference,
// it will use credentials if supplied
func OrasPush(file, ref string, ociAuth *ocitypes.DockerAuthConfig) error {
	ref = strings.TrimPrefix(ref, "//")

	spec, err := reference.Parse(ref)
	if err != nil {
		return fmt.Errorf("unable to parse oci reference: %s", err)
	}

	// Hostname() will panic if there is no '/' in the locator
	// explicitly check for this and fail in order to prevent panic
	// this case will only occur for incorrect uris
	if !strings.Contains(spec.Locator, "/") {
		return fmt.Errorf("not a valid oci object uri: %s", ref)
	}

	// append default tag if no object exists
	if spec.Object == "" {
		spec.Object = SifDefaultTag
		sylog.Infof("No tag or digest found, using default: %s", SifDefaultTag)
	}

	credFn := func(_ string) (string, string, error) {
		return ociAuth.Username, ociAuth.Password, nil
	}

	resolver := docker.NewResolver(docker.ResolverOptions{Credentials: credFn})

	store := content.NewFileStore("")
	defer store.Close()

	conf, err := store.Add("$config", SifConfigMediaType, "/dev/null")
	if err != nil {
		return fmt.Errorf("unable to add manifest config to FileStore: %s", err)
	}
	conf.Annotations = nil

	desc, err := store.Add(file, SifLayerMediaType, file)
	if err != nil {
		return fmt.Errorf("unable to add SIF file to FileStore: %s", err)
	}

	descriptors := []ocispec.Descriptor{desc}

	if _, err := oras.Push(context.Background(), resolver, spec.String(), store, descriptors, oras.WithConfig(conf)); err != nil {
		return fmt.Errorf("unable to push: %s", err)
	}

	return nil
}

// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// TODO(ian): The build package should be refactored to make each conveyorpacker
// its own separate package. With that change, this file should be grouped with the
// OCIConveyorPacker code

package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/containers/image/types"
	"github.com/openSUSE/umoci"
	umocilayer "github.com/openSUSE/umoci/oci/layer"
	"github.com/openSUSE/umoci/pkg/idtools"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	sytypes "github.com/sylabs/singularity/pkg/build/types"
)

// unpackRootfs extracts all of the layers of the given image reference into the rootfs of the provided bundle
func unpackRootfs(b *sytypes.Bundle, tmpfsRef types.ImageReference, sysCtx *types.SystemContext) (err error) {
	var mapOptions umocilayer.MapOptions

	// Allow unpacking as non-root
	if os.Geteuid() != 0 {
		mapOptions.Rootless = true

		uidMap, err := idtools.ParseMapping(fmt.Sprintf("0:%d:1", os.Geteuid()))
		if err != nil {
			return fmt.Errorf("error parsing uidmap: %s", err)
		}
		mapOptions.UIDMappings = append(mapOptions.UIDMappings, uidMap)

		gidMap, err := idtools.ParseMapping(fmt.Sprintf("0:%d:1", os.Getegid()))
		if err != nil {
			return fmt.Errorf("error parsing gidmap: %s", err)
		}
		mapOptions.GIDMappings = append(mapOptions.GIDMappings, gidMap)
	}

	engineExt, err := umoci.OpenLayout(b.Path)
	if err != nil {
		return fmt.Errorf("error opening layout: %s", err)
	}

	// Obtain the manifest
	imageSource, err := tmpfsRef.NewImageSource(context.Background(), sysCtx)
	if err != nil {
		return fmt.Errorf("error creating image source: %s", err)
	}
	manifestData, mediaType, err := imageSource.GetManifest(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("error obtaining manifest source: %s", err)
	}
	if mediaType != imgspecv1.MediaTypeImageManifest {
		return fmt.Errorf("error verifying manifest media type: %s", mediaType)
	}
	var manifest imgspecv1.Manifest
	json.Unmarshal(manifestData, &manifest)

	// UnpackRootfs from umoci v0.4.2 expects a path to a non-existing directory
	os.RemoveAll(b.Rootfs())

	// Unpack root filesystem
	return umocilayer.UnpackRootfs(context.Background(), engineExt, b.Rootfs(), manifest, &mapOptions)
}

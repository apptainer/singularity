// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// TODO(ian): The build package should be refactored to make each conveyorpacker
// its own separate package. With that change, this file should be grouped with the
// OCIConveyorPacker code

package sources

import (
	"github.com/containers/image/types"
	imagetools "github.com/opencontainers/image-tools/image"
	sytypes "github.com/sylabs/singularity/pkg/build/types"
)

// unpackRootfs extracts all of the layers of the given image reference into the rootfs of the provided bundle
func unpackRootfs(b *sytypes.Bundle, _ types.ImageReference, _ *types.SystemContext) (err error) {
	refs := []string{"name=tmp"}
	err = imagetools.UnpackLayout(b.Path, b.Rootfs(), "amd64", refs)
	return err
}

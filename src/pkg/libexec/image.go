// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package libexec

import (
	image "github.com/singularityware/singularity/src/pkg/image"
	//specs "github.com/opencontainers/runtime-spec/specs-go"
)

// ImageCreate creates an image.
func ImageCreate() image.Image {
	return image.SandboxFromPath("/path/to/sandbox")
}

// ImageBuild builds an image.
func ImageBuild() image.Image {
	return image.SandboxFromPath("/path/to/sandbox")
}

// ImageExpand expands an image.
func ImageExpand() image.Image {
	return image.SandboxFromPath("/path/to/sandbox")
}

// ImageImport imports an image.
func ImageImport() {

}

// ImageExport exports an image.
func ImageExport() {

}

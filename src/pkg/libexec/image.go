/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package libexec

import (
	image "github.com/singularityware/singularity/pkg/image"
	//specs "github.com/opencontainers/runtime-spec/specs-go"
)

func ImageCreate() image.Image {
	return image.SandboxFromPath("/path/to/sandbox")
}

func ImageBuild() image.Image {
	return image.SandboxFromPath("/path/to/sandbox")
}

func ImageExpand() image.Image {
	return image.SandboxFromPath("/path/to/sandbox")
}

func ImageImport() {

}

func ImageExport() {

}

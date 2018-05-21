// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"context"

	"github.com/singularityware/singularity/src/pkg/image"
	"github.com/singularityware/singularity/src/pkg/sif"
)

// LocalBuilder is an interface that enables building an image using a local
// sandbox.
type LocalBuilder struct {
	Sandbox image.Sandbox
	Image   image.Image
	Definition
}

// NewLocalBuilder creates a new LocalBuilder.
func NewLocalBuilder(j []byte) LocalBuilder {
	return LocalBuilder{image.Sandbox{}, &sif.SIF{}, DefinitionFromJSON(j)}
}

// Build completes a build. The supplied context can be used for cancellation.
func (*LocalBuilder) Build(ctx context.Context) {

}

// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"github.com/sylabs/singularity/pkg/build/types"
)

// Conveyor is responsible for downloading from remote sources (library, shub, docker...)
type Conveyor interface {
	Get(*types.Bundle) error
}

// Packer is the type which is responsible for installing the chroot directory,
// metadata directory, and potentially other files/directories within the Bundle
type Packer interface {
	Pack() (*types.Bundle, error)
}

// ConveyorPacker describes an interface that a ConveyorPacker type must implement
type ConveyorPacker interface {
	Conveyor
	Packer
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"fmt"
	"strings"
)

// validConveyorPackers contains a list of known ConveyorPackers
var validConveyorPackers = map[string]bool{
	"library":     true,
	"docker":      true,
	"shub":        true,
	"debootstrap": true,
	"yum":         true,
	"squashfs":    true,
	"sif":         true,
}

// Conveyor is responsible for downloading from remote sources (library, shub, docker...)
type Conveyor interface {
	Get(string) error
}

// Packer is the type which is responsible for installing the chroot directory,
// metadata directory, and potentially other files/directories within the Bundle
type Packer interface {
	Pack() (*Bundle, error)
}

// ConveyorPacker describes an interface that a ConveyorPacker type must implement
type ConveyorPacker interface {
	Conveyor
	Packer
}

// IsValidConveyorPacker returns whether or not the given source is valid
func IsValidConveyorPacker(source string) (valid bool, err error) {
	u := strings.SplitN(source, ":", 2)

	if len(u) != 2 {
		return false, nil
	}

	if _, ok := validConveyorPackers[u[0]]; ok {
		return true, nil
	}

	return false, fmt.Errorf("Invalid ConveyorPacker %s", source)
}

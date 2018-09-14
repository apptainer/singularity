// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"fmt"
	"strings"

	"github.com/singularityware/singularity/src/pkg/build/types"
)

// validURIs contains a list of known uris
var validURIs = map[string]bool{
	"shub":           true,
	"docker":         true,
	"docker-archive": true,
	"docker-daemon":  true,
	"oci":            true,
	"oci-archive":    true,
}

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

// IsValidURI returns whether or not the given source is valid
func IsValidURI(source string) (valid bool, err error) {

	u := strings.SplitN(source, ":", 2)

	if len(u) != 2 {
		return false, fmt.Errorf("Invalid URI %s", source)
	}

	if _, ok := validURIs[u[0]]; ok {
		return true, nil
	}

	return false, fmt.Errorf("Invalid URI %s", source)
}

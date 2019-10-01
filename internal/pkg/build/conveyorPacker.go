// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"context"
	"fmt"

	"github.com/sylabs/singularity/internal/pkg/build/sources"
	"github.com/sylabs/singularity/pkg/build/types"
)

// Conveyor is responsible for downloading from remote sources (library, shub, docker...).
type Conveyor interface {
	Get(context.Context, *types.Bundle) error
}

// Packer is the type which is responsible for installing the chroot directory,
// metadata directory, and potentially other files/directories within the Bundle.
type Packer interface {
	Pack(context.Context) (*types.Bundle, error)
}

// ConveyorPacker describes an interface that a ConveyorPacker type must implement.
type ConveyorPacker interface {
	Conveyor
	Packer
}

// conveyorPacker returns a valid ConveyorPacker for the given image definition.
func conveyorPacker(def types.Definition) (ConveyorPacker, error) {
	switch def.Header["bootstrap"] {
	case "library":
		return &sources.LibraryConveyorPacker{}, nil
	case "oras":
		return &sources.OrasConveyorPacker{}, nil
	case "shub":
		return &sources.ShubConveyorPacker{}, nil
	case "docker", "docker-archive", "docker-daemon", "oci", "oci-archive":
		return &sources.OCIConveyorPacker{}, nil
	case "busybox":
		return &sources.BusyBoxConveyorPacker{}, nil
	case "debootstrap":
		return &sources.DebootstrapConveyorPacker{}, nil
	case "arch":
		return &sources.ArchConveyorPacker{}, nil
	case "localimage":
		return &sources.LocalConveyorPacker{}, nil
	case "yum":
		return &sources.YumConveyorPacker{}, nil
	case "zypper":
		return &sources.ZypperConveyorPacker{}, nil
	case "scratch":
		return &sources.ScratchConveyorPacker{}, nil
	case "":
		return nil, fmt.Errorf("no bootstrap specification found")
	default:
		return nil, fmt.Errorf("invalid build source %s", def.Header["bootstrap"])
	}
}

// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"fmt"

	"github.com/sylabs/singularity/pkg/runtime/engine/config"
)

// DriverFeature defines a feature type that a driver is supporting.
type DriverFeature uint16

const (
	// ImageFeature means the driver handle image mount setup.
	ImageFeature DriverFeature = 1 << iota
	// OverlayFeature means the driver handle overlay mount.
	OverlayFeature
	// FuseFeature means the driver use FUSE.
	FuseFeature
)

// MountFunc defines mount function prototype
type MountFunc func(source string, target string, filesystem string, flags uintptr, data string) error

// MountParams defines parameters passed to driver interface
// while mounting images.
type MountParams struct {
	Source     string   // image source
	Target     string   // image target mount point
	Filesystem string   // image filesystem type
	Flags      uintptr  // mount flags
	Offset     uint64   // offset where start filesystem
	Size       uint64   // size of image filesystem
	Key        []byte   // filesystem decryption key
	FSOptions  []string // filesystem mount options
}

// DriverParams defines parameters passed to driver interface
// while starting it.
type DriverParams struct {
	SessionPath string         // session driver image path
	UsernsFd    int            // user namespace file descriptor
	FuseFd      int            // fuse file descriptor
	Config      *config.Common // common engine configuration
}

// Driver defines the image driver interface to register.
type Driver interface {
	// Mount is called each time an engine mount an image
	Mount(*MountParams, MountFunc) error
	// Start the driver for initialization.
	Start(*DriverParams) error
	// Stop the driver for cleanup.
	Stop() error
	// Feature returns supported features.
	Features() DriverFeature
}

// drivers holds all registered image drivers
var drivers = make(map[string]Driver)

// RegisterDriver registers an image driver by name.
func RegisterDriver(name string, driver Driver) error {
	if name == "" {
		return fmt.Errorf("empty name")
	} else if _, ok := drivers[name]; ok {
		return fmt.Errorf("%s is already registered", name)
	} else if driver == nil {
		return fmt.Errorf("nil driver")
	}
	drivers[name] = driver
	return nil
}

// GetDriver returns the named image driver interface.
func GetDriver(name string) Driver {
	return drivers[name]
}

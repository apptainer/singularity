// Copyright (c) 2019-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package overlay

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// statfs is the function pointing to unix.Statfs and
// also used by unit tests for mocking.
var statfs = unix.Statfs

type dir uint8

const (
	_ dir = iota << 1
	lowerDir
	upperDir
)

type fs struct {
	name       string
	overlayDir dir
}

const (
	nfs    int64 = 0x6969
	fuse   int64 = 0x65735546
	ecrypt int64 = 0xF15F
	lustre int64 = 0x0BD00BD0 //nolint:misspell
	gpfs   int64 = 0x47504653
)

var incompatibleFs = map[int64]fs{
	// NFS filesystem
	nfs: {
		name:       "NFS",
		overlayDir: upperDir,
	},
	// FUSE filesystem
	fuse: {
		name:       "FUSE",
		overlayDir: upperDir,
	},
	// ECRYPT filesystem
	ecrypt: {
		name:       "ECRYPT",
		overlayDir: lowerDir | upperDir,
	},
	// LUSTRE filesystem
	//nolint:misspell
	lustre: {
		name:       "LUSTRE",
		overlayDir: lowerDir | upperDir,
	},
	// GPFS filesystem
	gpfs: {
		name:       "GPFS",
		overlayDir: lowerDir | upperDir,
	},
}

func check(path string, d dir) error {
	stfs := &unix.Statfs_t{}

	if err := statfs(path, stfs); err != nil {
		return fmt.Errorf("could not retrieve underlying filesystem information for %s: %s", path, err)
	}

	fs, ok := incompatibleFs[int64(stfs.Type)]
	if !ok || (ok && fs.overlayDir&d == 0) {
		return nil
	}

	return &errIncompatibleFs{
		path: path,
		name: fs.name,
		dir:  d,
	}
}

// CheckUpper checks if the underlying filesystem of the
// provided path can be used as an upper overlay directory.
func CheckUpper(path string) error {
	return check(path, upperDir)
}

// CheckLower checks if the underlying filesystem of the
// provided path can be used as lower overlay directory.
func CheckLower(path string) error {
	return check(path, lowerDir)
}

type errIncompatibleFs struct {
	path string
	name string
	dir  dir
}

func (e *errIncompatibleFs) Error() string {
	overlayDir := "lower"
	if e.dir == upperDir {
		overlayDir = "upper"
	}
	return fmt.Sprintf(
		"%s is located on a %s filesystem incompatible as overlay %s directory",
		e.path, e.name, overlayDir,
	)
}

// IsIncompatible returns if the error corresponds to
// an incompatible filesystem error.
func IsIncompatible(err error) bool {
	if _, ok := err.(*errIncompatibleFs); ok {
		return true
	}
	return false
}

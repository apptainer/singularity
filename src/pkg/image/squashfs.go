// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// SquashFS represents a SquashFS image
type SquashFS struct {
}

// SquashFSFromSandbox converts the sandbox, s, to a SquashFS file
func SquashFSFromSandbox(s *Sandbox) *SquashFS {
	return &SquashFS{}
}

// SquashFSFromFile returns a squashfs object of the file located at path
func SquashFSFromFile(path string) *SquashFS {
	return &SquashFS{}
}

// Root returns the root specification.
func (i *SquashFS) Root() *specs.Root {
	return &specs.Root{}
}

// isSquashFs checks the "magic" of the given file and
// determines if the file is of squashfs type
func isSquashFs(path string) bool {
	return false
}

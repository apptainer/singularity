// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// EXT3 represents an EXT3 image
type EXT3 struct {
}

// Ext3FromSandbox converts the sandbox, s, to a EXT3 file
func Ext3FromSandbox(s *Sandbox) *EXT3 {
	return &EXT3{}
}

// Ext3FromPath returns a ext3 object of the file located at path
func Ext3FromPath(path string) *EXT3 {
	return &EXT3{}
}

// Root returns the root specification.
func (i *EXT3) Root() *specs.Root {
	return &specs.Root{}
}

// isExt3 checks the "magic" of the given file and
// determines if the file is of the ext3 type
func isExt3(path string) bool {
	return false
}

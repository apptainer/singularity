/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package image

import (
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

type squashfs struct {
}

// SquashFsFromSandbox converts the sandbox, s, to a SquashFS file
func SquashFsFromSandbox(s *Sandbox) *squashfs {
	return &squashfs{}
}

// SquashFsFromFile returns a squashfs object of the file located at path
func SquashFsFromFile(path string) *squashfs {
	return &squashfs{}
}

func (i *squashfs) Root() *specs.Root {
	return &specs.Root{}
}

// isSquashFs checks the "magic" of the given file and
// determines if the file is of squashfs type
func isSquashFs(path string) bool {
	return false
}

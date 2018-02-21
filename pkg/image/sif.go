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

type sif struct {
}

// SifFromSandbox converts the sandbox, s, to a SIF file
func SifFromSandbox(s *Sandbox) *sif {
	return &sif{}
}

// SifFromPath returns a sif object of the file located at path
func SifFromPath(path string) *sif {
	return &sif{}
}

func (i *sif) Root() *specs.Root {
	return &specs.Root{}
}

// isSif checks the "magic" of the given file and
// determines if the file is of the sif type
func isSif(path string) bool {
	return false
}

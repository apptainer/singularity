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

type SIF struct {
}

// SIFFromSandbox converts the sandbox, s, to a SIF file
func SIFFromSandbox(s *Sandbox) *SIF {
	return &SIF{}
}

// SIFFromPath returns a SIF object of the file located at path
func SIFFromPath(path string) *SIF {
	return &SIF{}
}

func (i *SIF) Root() *specs.Root {
	return &specs.Root{}
}

// isSIF checks the "magic" of the given file and
// determines if the file is of the SIF type
func isSIF(path string) bool {
	return false
}

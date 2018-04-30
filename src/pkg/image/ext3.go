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

type ext3 struct {
}

// Ext3FromSandbox converts the sandbox, s, to a EXT3 file
func Ext3FromSandbox(s *Sandbox) *ext3 {
	return &ext3{}
}

// Ext3FromPath returns a ext3 object of the file located at path
func Ext3FromPath(path string) *ext3 {
	return &ext3{}
}

func (i *ext3) Root() *specs.Root {
	return &specs.Root{}
}

// isExt3 checks the "magic" of the given file and
// determines if the file is of the ext3 type
func isExt3(path string) bool {
	return false
}

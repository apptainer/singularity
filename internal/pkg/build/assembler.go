// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"fmt"

	"github.com/sylabs/singularity/pkg/build/types"
)

// validAssemblers contains of list of know Assemblers
var validAssemblers = map[string]bool{
	"SIF":     true,
	"sandbox": true,
}

// Assembler is responsible for assembling an image from a bundle.
// For example a bundle may be holding multiple file systems indended
// to be separate partitions within a SIF image. The assembler would need
// to detect these directories and make sure it properly assembles the SIF
// with them as partitions
type Assembler interface {
	Assemble(*types.Bundle, string) error
}

// IsValidAssembler returns whether or not the given Assembler is valid
func IsValidAssembler(c string) (valid bool, err error) {
	if _, ok := validAssemblers[c]; ok {
		return true, nil
	}

	return false, fmt.Errorf("Invalid Assembler %s", c)
}

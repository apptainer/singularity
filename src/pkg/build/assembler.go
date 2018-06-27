// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"fmt"
)

// validAssemblers contains of list of know Assemblers
var validAssemblers = map[string]bool{
	"SIF":     true,
	"sandbox": true,
}

// Assembler is responsible for assembling an image from a bundle
type Assembler interface {
	Assemble(*Bundle, string) error
}

// IsValidAssembler returns whether or not the given Assembler is valid
func IsValidAssembler(c string) (valid bool, err error) {
	if _, ok := validAssemblers[c]; ok {
		return true, nil
	}

	return false, fmt.Errorf("Invalid Assembler %s", c)
}

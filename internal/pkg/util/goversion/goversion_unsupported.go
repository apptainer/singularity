// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !go1.10

package goversion

import "fmt"

// Check checks if we are running with the minimal recommended version of Go
func Check() error {
	return fmt.Errorf("Singularity requires to be compiled with a Go version >= 1.10")
}

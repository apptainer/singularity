// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package libexec

import (
	"fmt"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// Shell drops to a shell.
func Shell(spec *specs.Spec) {
	fmt.Println("Shell")
}

// Exec executes the supplied command.
func Exec(spec *specs.Spec, cmd string) {

}

// Run runs the image.
func Run(spec *specs.Spec) {

}

// Test tests the image.
func Test(spec *specs.Spec) {

}

// SelfTest runs a self-test.
func SelfTest(spec *specs.Spec) {

}

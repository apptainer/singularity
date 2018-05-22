// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package libexec

import (
	"fmt"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func Shell(spec *specs.Spec) {
	fmt.Println("Shell")
}

func Exec(spec *specs.Spec, cmd string) {

}

func Run(spec *specs.Spec) {

}

func Test(spec *specs.Spec) {

}

func SelfTest(spec *specs.Spec) {

}

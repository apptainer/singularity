// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !go1.10 !linux

package namespaces

import (
	"fmt"
	"runtime"
)

// Enter enters in provided process namespace.
func Enter(pid int, namespace string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("%s system is unsupported", runtime.GOOS)
	}
	return fmt.Errorf("using setns requires a compilation with Go version >= 1.10")
}

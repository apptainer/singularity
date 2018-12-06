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
		return fmt.Errorf("unsupported platform")
	}
	return fmt.Errorf("was compiled with go version < 1.10")
}

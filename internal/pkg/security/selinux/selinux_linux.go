// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build selinux

package selinux

import (
	goselinux "github.com/opencontainers/selinux/go-selinux"
)

// Enabled checks if SELinux is enabled or not
func Enabled() bool {
	return goselinux.GetEnabled()
}

// SetExecLabel sets the SELinux label for current process
func SetExecLabel(label string) error {
	return goselinux.SetExecLabel(label)
}

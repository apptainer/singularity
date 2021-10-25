// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

//go:build !selinux
// +build !selinux

package selinux

import "errors"

// Enabled returns whether SELinux is enabled.
func Enabled() bool {
	return false
}

// SetExecLabel sets the SELinux label for current process.
func SetExecLabel(label string) error {
	return errors.New("can't set SELinux label: not enabled at compilation time")
}

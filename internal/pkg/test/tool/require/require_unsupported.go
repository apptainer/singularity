// Copyright (c) 2019,2020 Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !linux

package require

import "testing"

// UserNamespace checks that the current test could use
// user namespace, if user namespaces are not enabled or
// supported, the current test is skipped with a message.
func UserNamespace(t *testing.T) {
	t.Skipf("user namespaces not supported on this platform")
}

// Network check that bridge network is supported by
// system, if not the current test is skipped with a
// message.
func Network(t *testing.T) {
	t.Skipf("network not supported on this platform")
}

// Cgroups checks that cgroups is enabled, if not the
// current test is skipped with a message.
func Cgroups(t *testing.T) {
	t.Skipf("cgroups not supported on this platform")
}

// CgroupsFreezer checks that cgroup freezer subsystem is
// available, if not the current test is skipped with a
// message
func CgroupsFreezer(t *testing.T) {
	t.Skipf("cgroups not supported on this platform")
}

// Nvidia checks that an NVIDIA stack is available
func Nvidia(t *testing.T) {
	t.Skipf("nvidia not supported on this platform")
}

// Rocm checks that a Rocm stack is available
func Rocm(t *testing.T) {
	t.Skipf("rocm not supported on this platform")
}

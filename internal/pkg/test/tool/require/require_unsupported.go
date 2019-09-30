// Copyright (c) 2019, Sylabs Inc. All rights reserved.
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

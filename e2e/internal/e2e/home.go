// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !linux

package e2e

import "testing"

// SetupHomeDirectories creates temporary home directories for
// privileged and unprivileged users and bind mount those directories
// on top of real ones. It's possible because e2e tests are executed
// in a dedicated mount namespace.
func SetupHomeDirectories(t *testing.T) {
}

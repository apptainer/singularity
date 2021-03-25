// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !linux

package e2e

import "testing"

func SetupSystemRemoteFile(t *testing.T, testDir string) {
	// TODO - address unsupported tests on MacOS better
	// This is a hack to avoid golangci-lint failures on non-Linux
	t.Fatalf("Remote tests only supported on Linux")
}

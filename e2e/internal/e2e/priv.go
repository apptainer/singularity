// Copyright (c) 2019,2020 Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"os"
)

var (
	// uid of original user running test.
	origUID = os.Getuid()
	// gid of original group running test.
	origGID = os.Getgid()
)

// OrigUID returns the UID of the user running the test suite.
func OrigUID() int {
	return origUID
}

// OrigGID returns the GID of the user running the test suite.
func OrigGID() int {
	return origGID
}

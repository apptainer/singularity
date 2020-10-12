// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"testing"
)

// Privileged on Darwin will not actually raise privileges
func Privileged(f func(*testing.T)) func(*testing.T) {
	return f
}

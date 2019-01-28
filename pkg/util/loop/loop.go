// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package loop

import (
	"os"
)

// Device describes a loop device
type Device struct {
	MaxLoopDevices int
	Shared         bool
	Info           *Info64
	file           *os.File
}

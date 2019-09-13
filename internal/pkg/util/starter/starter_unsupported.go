// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !linux

package starter

import (
	"fmt"
)

// sendData sets a socket communication channel between caller and starter
// binary in order to pass engine JSON configuration data to starter.
func sendData(data []byte) (int, error) {
	return -1, fmt.Errorf("not supported on this platform")
}

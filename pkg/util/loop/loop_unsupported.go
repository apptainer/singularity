// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !linux

package loop

import (
	"fmt"
	"os"
)

// AttachFromFile finds a free loop device, opens it, and stores file descriptor
// provided by image file pointer
func (loop *Device) AttachFromFile(image *os.File, mode int, number *int) error {
	return fmt.Errorf("unsupported on this platform")
}

// AttachFromPath finds a free loop device, opens it, and stores file descriptor
// of opened image path
func (loop *Device) AttachFromPath(image string, mode int, number *int) error {
	return fmt.Errorf("unsupported on this platform")
}

// GetStatusFromFd gets info status about an opened loop device
func GetStatusFromFd(fd uintptr) (*Info64, error) {
	return nil, fmt.Errorf("unsupported on this platform")
}

// GetStatusFromPath gets info status about a loop device from path
func GetStatusFromPath(path string) (*Info64, error) {
	return nil, fmt.Errorf("unsupported on this platform")
}

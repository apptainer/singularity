// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !linux

package capabilities

import (
	"fmt"
	"runtime"
)

var ErrCapNotSupported = fmt.Errorf("capabilities not supported on this OS: %s", runtime.GOOS)

// GetProcessEffective returns effective capabilities for
// the current process.
func GetProcessEffective() (uint64, error) {
	return 0, ErrCapNotSupported
}

// GetProcessPermitted returns permitted capabilities for
// the current process.
func GetProcessPermitted() (uint64, error) {
	return 0, ErrCapNotSupported
}

// GetProcessInheritable returns inheritable capabilities for
// the current process.
func GetProcessInheritable() (uint64, error) {
	return 0, ErrCapNotSupported
}

// SetProcessEffective set effective capabilities for the
// the current process and returns previous effective set.
func SetProcessEffective(caps uint64) (uint64, error) {
	return 0, ErrCapNotSupported
}

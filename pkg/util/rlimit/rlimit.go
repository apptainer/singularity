// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !linux

package rlimit

import (
	"fmt"
)

// Set sets soft and hard resource limit
func Set(res string, cur uint64, max uint64) error {
	return fmt.Errorf("not supported on this platform")
}

// Get retrieves soft and hard resource limit
func Get(res string) (cur uint64, max uint64, err error) {
	return 0, 0, fmt.Errorf("not supported on this platform")
}

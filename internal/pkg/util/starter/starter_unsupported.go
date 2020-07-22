// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !linux

package starter

import (
	"fmt"
)

// copyConfigToEnv checks that the current stack size is big enough
// to pass runtime configuration through environment variables.
// On linux RLIMIT_STACK determines the amount of space used for the
// process's command-line arguments and environment variables.
func copyConfigToEnv(data []byte) ([]string, error) {
	return nil, fmt.Errorf("not supported on this platform")
}

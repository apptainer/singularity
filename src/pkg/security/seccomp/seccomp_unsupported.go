// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !seccomp OR !linux

package seccomp

import (
	"fmt"
	"runtime"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// LoadSeccompConfig returns an error for unsupported platforms or without seccomp support
func LoadSeccompConfig(config *specs.LinuxSeccomp) error {
	if runtime.GOOS == "linux" {
		return fmt.Errorf("can't load seccomp filter: not enabled at compilation time")
	}
	return fmt.Errorf("can't load seccomp filter: not supported by OS")
}

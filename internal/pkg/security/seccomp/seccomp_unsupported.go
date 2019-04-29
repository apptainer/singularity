// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !seccomp OR !linux

package seccomp

import (
	"fmt"
	"runtime"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
)

// Enabled returns wether seccomp is enabled or not
func Enabled() bool {
	return false
}

// LoadSeccompConfig returns an error for unsupported platforms or without seccomp support
func LoadSeccompConfig(config *specs.LinuxSeccomp, noNewPrivs bool) error {
	if runtime.GOOS == "linux" {
		return fmt.Errorf("can't load seccomp filter: not enabled at compilation time")
	}
	return fmt.Errorf("can't load seccomp filter: not supported by OS")
}

// LoadProfileFromFile sets an empty seccomp configuration for unsupported platforms
func LoadProfileFromFile(profile string, generator *generate.Generator) error {
	if generator.Config.Linux == nil {
		generator.Config.Linux = &specs.Linux{}
	}
	if generator.Config.Linux.Seccomp == nil {
		generator.Config.Linux.Seccomp = &specs.LinuxSeccomp{}
	}
	return nil
}

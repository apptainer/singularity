// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !seccomp

package seccomp

import (
	"fmt"

	"github.com/hpcng/singularity/internal/pkg/runtime/engine/config/oci/generate"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// Enabled returns whether seccomp is enabled.
func Enabled() bool {
	return false
}

// LoadSeccompConfig loads seccomp configuration filter for the current process.
func LoadSeccompConfig(config *specs.LinuxSeccomp, noNewPrivs bool, errNo int16) error {
	return fmt.Errorf("can't load seccomp filter: not enabled at compilation time")
}

// LoadProfileFromFile loads seccomp rules from json file and fill in provided OCI configuration.
func LoadProfileFromFile(profile string, generator *generate.Generator) error {
	if generator.Config.Linux == nil {
		generator.Config.Linux = &specs.Linux{}
	}
	if generator.Config.Linux.Seccomp == nil {
		generator.Config.Linux.Seccomp = &specs.LinuxSeccomp{}
	}
	return nil
}

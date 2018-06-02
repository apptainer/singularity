// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"github.com/opencontainers/runtime-spec/specs-go"
)

// RuntimeOciSpec is the OCI runtime specification.
type RuntimeOciSpec specs.Spec

// RuntimeOciConfig is the OCI runtime configuration.
type RuntimeOciConfig struct {
	RuntimeOciSpec
	Version     RuntimeOciVersion
	Process     RuntimeOciProcess
	Root        RuntimeOciRoot
	Hostname    RuntimeOciHostname
	Mounts      RuntimeOciMounts
	Hooks       RuntimeOciHooks
	Annotations RuntimeOciAnnotations
	RuntimeOciPlatform
}

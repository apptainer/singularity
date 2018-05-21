// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"github.com/opencontainers/runtime-spec/specs-go"
)

type RuntimeOciMounts interface {
	GetSpec() *specs.Mount

	GetMounts() []specs.Mount
	SetMounts(mounts []specs.Mount) error
	AddMount(destination string, mounttype string, source string, options []string) error
	DelMount(destination string) error
}

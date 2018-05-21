// +build linux solaris

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"github.com/opencontainers/runtime-spec/specs-go"
)

type RuntimeOciHooks interface {
	GetSpec() *specs.Hook

	GetPrestartHook() []specs.Hook
	SetPrestartHook(hooks []specs.Hook) error
	AddPrestartHook(path string, args []string, env []string, timeout *int) error
	DelPrestartHook(path string) error

	GetPoststartHook() []specs.Hook
	SetPoststartHook(hooks []specs.Hook) error
	AddPoststartHook(path string, args []string, env []string, timeout *int) error
	DelPoststartHook(path string) error

	GetPoststopHook() []specs.Hook
	SetPoststopHook(hooks []specs.Hook) error
	AddPoststopHook(path string, args []string, env []string, timeout *int) error
	DelPoststopHook(path string) error
}

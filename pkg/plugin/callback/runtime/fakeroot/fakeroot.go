// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package fakeroot

import (
	"github.com/opencontainers/runtime-spec/specs-go"
)

// UserMapping callback returns fakeroot user mappings from plugin
// (eg: to get fakeroot mapping from an external database). If more
// than one plugin uses this callback the runtime aborts its execution.
// This callback is called in:
// - internal/pkg/runtime/engine/fakeroot/engine_linux.go (build command)
// - internal/pkg/runtime/engine/singularity/prepare_linux.go (actions commands)
// This function is usually called two times, a first time with path
// set to "/etc/subuid" and a second time with path set to "/etc/subgid"
// to get container UID and GID mappings for the user specified by the
// uid parameter.
type UserMapping func(path string, uid uint32) (*specs.LinuxIDMapping, error)

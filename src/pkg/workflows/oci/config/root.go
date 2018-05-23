// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"github.com/opencontainers/runtime-spec/specs-go"
)

// RuntimeOciRoot describes the methods required for an OCI root implementation.
type RuntimeOciRoot interface {
	GetSpec() *specs.Root

	GetPath() string
	SetPath(path string)

	GetReadOnly() bool
	SetReadOnly(enabled bool)
}

// DefaultRuntimeOciRoot describes the default runtime OCI root.
type DefaultRuntimeOciRoot struct {
	RuntimeOciSpec *RuntimeOciSpec
}

func (c *DefaultRuntimeOciRoot) init() {
	if c.RuntimeOciSpec.Root == nil {
		c.RuntimeOciSpec.Root = &specs.Root{}
	}
}

// GetSpec retrieves the runtime OCI root spec.
func (c *DefaultRuntimeOciRoot) GetSpec() *specs.Root {
	c.init()
	return c.RuntimeOciSpec.Root
}

// GetPath retrieves the runtime OCI root path.
func (c *DefaultRuntimeOciRoot) GetPath() string {
	c.init()
	return c.RuntimeOciSpec.Root.Path
}

// SetPath sets the runtime OCI root path.
func (c *DefaultRuntimeOciRoot) SetPath(path string) {
	c.init()
	c.RuntimeOciSpec.Root.Path = path
}

// GetReadOnly gets the runtime OCI root read-only flag.
func (c *DefaultRuntimeOciRoot) GetReadOnly() bool {
	c.init()
	return c.RuntimeOciSpec.Root.Readonly
}

// SetReadOnly sets the runtime OCI root read-only flag.
func (c *DefaultRuntimeOciRoot) SetReadOnly(enabled bool) {
	c.init()
	c.RuntimeOciSpec.Root.Readonly = enabled
}

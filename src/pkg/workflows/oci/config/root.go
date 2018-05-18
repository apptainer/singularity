/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package config

import (
	"github.com/opencontainers/runtime-spec/specs-go"
)

type RuntimeOciRoot interface {
	GetSpec() *specs.Root

	GetPath() string
	SetPath(path string)

	GetReadOnly() bool
	SetReadOnly(enabled bool)
}

type DefaultRuntimeOciRoot struct {
	RuntimeOciSpec *RuntimeOciSpec
}

func (c *DefaultRuntimeOciRoot) init() {
	if c.RuntimeOciSpec.Root == nil {
		c.RuntimeOciSpec.Root = &specs.Root{}
	}
}

func (c *DefaultRuntimeOciRoot) GetSpec() *specs.Root {
	c.init()
	return c.RuntimeOciSpec.Root
}

func (c *DefaultRuntimeOciRoot) GetPath() string {
	c.init()
	return c.RuntimeOciSpec.Root.Path
}

func (c *DefaultRuntimeOciRoot) SetPath(path string) {
	c.init()
	c.RuntimeOciSpec.Root.Path = path
}

func (c *DefaultRuntimeOciRoot) GetReadOnly() bool {
	c.init()
	return c.RuntimeOciSpec.Root.Readonly
}

func (c *DefaultRuntimeOciRoot) SetReadOnly(enabled bool) {
	c.init()
	c.RuntimeOciSpec.Root.Readonly = enabled
}

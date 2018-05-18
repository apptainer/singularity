/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package config

import (
	"fmt"
)

type RuntimeOciVersion interface {
	Get() string
	Set(name string)
}

type DefaultRuntimeOciVersion struct {
	RuntimeOciSpec *RuntimeOciSpec
}

func (c *DefaultRuntimeOciVersion) Get() string {
	fmt.Println("Get version")
	return c.RuntimeOciSpec.Version
}

func (c *DefaultRuntimeOciVersion) Set(version string) {
	fmt.Println("Set version to", version)
	c.RuntimeOciSpec.Version = version
}

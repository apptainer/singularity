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

type RuntimeOciHostname interface {
	Get() string
	Set(hostname string)
}

type DefaultRuntimeOciHostname struct {
	RuntimeOciSpec *RuntimeOciSpec
}

func (c *DefaultRuntimeOciHostname) Get() string {
	fmt.Println("Get hostname")
	return c.RuntimeOciSpec.Hostname
}

func (c *DefaultRuntimeOciHostname) Set(hostname string) {
	fmt.Println("Set hostname to", hostname)
	c.RuntimeOciSpec.Hostname = hostname
}

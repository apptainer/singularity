// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"fmt"
)

// RuntimeOciHostname describes the methods required for an OCI hostname implementation.
type RuntimeOciHostname interface {
	Get() string
	Set(hostname string)
}

// DefaultRuntimeOciHostname describes the default runtime OCI hostname.
type DefaultRuntimeOciHostname struct {
	RuntimeOciSpec *RuntimeOciSpec
}

// Get retrieves the hostname.
func (c *DefaultRuntimeOciHostname) Get() string {
	fmt.Println("Get hostname")
	return c.RuntimeOciSpec.Hostname
}

// Set sets the hostname.
func (c *DefaultRuntimeOciHostname) Set(hostname string) {
	fmt.Println("Set hostname to", hostname)
	c.RuntimeOciSpec.Hostname = hostname
}

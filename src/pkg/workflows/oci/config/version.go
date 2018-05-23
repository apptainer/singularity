// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"fmt"
)

// RuntimeOciVersion describes the OCI version interface.
type RuntimeOciVersion interface {
	Get() string
	Set(name string)
}

// DefaultRuntimeOciVersion describes the default runtime OCI version.
type DefaultRuntimeOciVersion struct {
	RuntimeOciSpec *RuntimeOciSpec
}

// Get retrieves the runtime OCI version.
func (c *DefaultRuntimeOciVersion) Get() string {
	fmt.Println("Get version")
	return c.RuntimeOciSpec.Version
}

// Set sets the runtime OCI version.
func (c *DefaultRuntimeOciVersion) Set(version string) {
	fmt.Println("Set version to", version)
	c.RuntimeOciSpec.Version = version
}

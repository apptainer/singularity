// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// Command describes the interface for a compliant OCI runtime
type Command interface {
	State(id string) *specs.State
	Create(id string, bundle string)
	Start(id string)
	Kill(id string, signal int)
	Delete(id string)
}

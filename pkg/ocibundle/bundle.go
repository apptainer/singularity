// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package ocibundle

import (
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// Bundle defines an OCI bundle interface to create/delete OCI bundles
type Bundle interface {
	Create(*specs.Spec) error
	Delete() error
}

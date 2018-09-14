// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package libexec

import (
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// InstanceStart starts an instance.
func InstanceStart(spec *specs.Spec) {

}

// InstanceStop stops an instance.
func InstanceStop(spec *specs.Spec) {

}

// InstanceList lists the running instances.
func InstanceList() []specs.State {
	return nil
}

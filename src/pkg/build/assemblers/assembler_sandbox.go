// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package assemblers

import (
	"os"

	"github.com/singularityware/singularity/src/pkg/build/types"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

// SandboxAssembler doesnt store anything
type SandboxAssembler struct {
}

// Assemble creates a Sandbox image from a Bundle
func (a *SandboxAssembler) Assemble(b *types.Bundle, path string) (err error) {
	defer os.RemoveAll(b.Path)

	//move bundle rootfs to sandboxdir as final sandbox
	sylog.Debugf("Moving sandbox from %v to %v", b.Rootfs(), path)
	if err := os.Rename(b.Rootfs(), path); err != nil {
		sylog.Errorf("Sandbox Assemble Failed", err.Error())
		return err
	}

	return nil
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package assemblers

import (
	"os"

	"github.com/otiai10/copy"
	"github.com/sylabs/singularity/src/pkg/build/types"
	"github.com/sylabs/singularity/src/pkg/sylog"
)

// SandboxAssembler doesnt store anything
type SandboxAssembler struct {
}

// Assemble creates a Sandbox image from a Bundle
func (a *SandboxAssembler) Assemble(b *types.Bundle, path string) (err error) {
	defer os.RemoveAll(b.Path)

	sylog.Infof("Creating sandbox directory...")

	// move bundle rootfs to sandboxdir as final sandbox
	sylog.Debugf("Moving sandbox from %v to %v", b.Rootfs(), path)
	if _, err := os.Stat(path); err == nil {
		os.RemoveAll(path)
	}
	if err := os.Rename(b.Rootfs(), path); err != nil {
		if err := copy.Copy(b.Rootfs(), path); err != nil {
			sylog.Errorf("Sandbox Assemble Failed: %s", err)
			return err
		}

		if err := os.RemoveAll(b.Rootfs()); err != nil {
			sylog.Errorf("Unable to remove Bundle directory: %s", err)
			return err
		}
	}

	return nil
}

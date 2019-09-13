// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package assemblers

import (
	"fmt"
	"os"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
)

// SandboxAssembler assembles a sandbox image.
type SandboxAssembler struct{}

// Assemble creates a Sandbox image from a Bundle.
func (a *SandboxAssembler) Assemble(b *types.Bundle, path string) (err error) {
	sylog.Infof("Creating sandbox directory...")

	sylog.Debugf("Moving sandbox from %v to %v", b.RootfsPath, path)
	if _, err := os.Stat(path); err == nil {
		os.RemoveAll(path)
	}

	err = os.Rename(b.RootfsPath, path)
	if err != nil {
		return fmt.Errorf("sandbox assemble failed: %v", err)
	}

	return nil
}

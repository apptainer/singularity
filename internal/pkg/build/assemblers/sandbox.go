// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package assemblers

import (
	"fmt"
	"os"

	"github.com/hpcng/singularity/pkg/build/types"
	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/hpcng/singularity/pkg/util/archive"
)

// SandboxAssembler assembles a sandbox image.
type SandboxAssembler struct {
	Copy bool
}

// Assemble creates a Sandbox image from a Bundle.
func (a *SandboxAssembler) Assemble(b *types.Bundle, path string) (err error) {
	sylog.Infof("Creating sandbox directory...")

	if _, err := os.Stat(path); err == nil {
		os.RemoveAll(path)
	}

	if a.Copy {
		sylog.Debugf("Copying sandbox from %v to %v", b.RootfsPath, path)

		err := archive.CopyWithTar(b.RootfsPath+`/.`, path)
		if err != nil {
			return fmt.Errorf("copy Failed: %v", err)
		}

	} else {
		sylog.Debugf("Moving sandbox from %v to %v", b.RootfsPath, path)

		err = os.Rename(b.RootfsPath, path)
		if err != nil {
			return fmt.Errorf("sandbox assemble failed: %v", err)
		}
	}

	return nil
}

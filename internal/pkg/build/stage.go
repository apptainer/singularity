// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
)

// stage represents the process of constucting a root filesystem
type stage struct {
	// name of the stage
	name string
	// c Gets and Packs data needed to build a container into a Bundle from various sources
	c ConveyorPacker
	// a Assembles a container from the information stored in a Bundle into various formats
	a Assembler
	// b is an intermediate structure that encapsulates all information for the container, e.g., metadata, filesystems
	b *types.Bundle
}

// Assemble assembles the bundle to the specified path
func (s *stage) Assemble(path string) error {
	return s.a.Assemble(s.b, path)
}

// runPreScript() executes the stages pre script on host
func (s *stage) runPreScript() error {
	if s.b.RunSection("pre") && s.b.Recipe.BuildData.Pre.Script != "" {
		if syscall.Getuid() != 0 {
			return fmt.Errorf("Attempted to build with scripts as non-root user")
		}

		// Run %pre script here
		pre := exec.Command("/bin/sh", "-cex", s.b.Recipe.BuildData.Pre.Script)
		pre.Stdout = os.Stdout
		pre.Stderr = os.Stderr

		sylog.Infof("Running pre scriptlet\n")
		if err := pre.Start(); err != nil {
			return fmt.Errorf("failed to start %%pre proc: %v", err)
		}
		if err := pre.Wait(); err != nil {
			return fmt.Errorf("pre proc: %v", err)
		}
	}
	return nil
}

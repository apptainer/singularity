// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
)

// Stage ...
type stage struct {
	// Name of the stage
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

// copyFiles allows a stage to copy files from the host to the bundle
func (s *stage) copyFiles() error {

	// iterate through files transfers
	for _, transfer := range s.b.Recipe.BuildData.Files {
		// sanity
		if transfer.Src == "" {
			sylog.Warningf("Attempt to copy file with no name...")
			continue
		}
		// dest = source if not specified
		if transfer.Dst == "" {
			transfer.Dst = transfer.Src
		}
		sylog.Infof("Copying %v to %v", transfer.Src, transfer.Dst)
		// copy each file into bundle rootfs
		transfer.Dst = filepath.Join(s.b.Rootfs(), transfer.Dst)
		copy := exec.Command("/bin/cp", "-fLr", transfer.Src, transfer.Dst)
		if err := copy.Run(); err != nil {
			return fmt.Errorf("While copying %v to %v: %v", transfer.Src, transfer.Dst, err)
		}
	}

	return nil
}

// runPreScript() executes the stages pre script on host
func (s *stage) runPreScript() error {
	if s.b.RunSection("pre") && s.b.Recipe.BuildData.Pre != "" {
		if syscall.Getuid() != 0 {
			return fmt.Errorf("Attempted to build with scripts as non-root user")
		}

		// Run %pre script here
		pre := exec.Command("/bin/sh", "-cex", s.b.Recipe.BuildData.Pre)
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

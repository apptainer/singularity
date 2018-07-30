// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package assemblers

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/singularityware/singularity/src/pkg/build/types"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

// SandboxAssembler doesnt store anything
type SandboxAssembler struct {
}

// Assemble creates a Sandbox image from a Bundle
func (a *SandboxAssembler) Assemble(b *types.Bundle, path string) (err error) {
	defer os.RemoveAll(b.Path)

	//make sandbox dir
	if err := os.MkdirAll(path, 0755); err != nil {
		sylog.Errorf("Making sandbox directory Failed", err.Error())
		return err
	}

	//copy bundle rootfs into sandboxdir
	cmd := exec.Command("cp", "-r", filepath.Join(b.Rootfs(), `/.`), path)
	err = cmd.Run()
	if err != nil {
		sylog.Errorf("Sandbox Assemble Failed", err.Error())
		return err
	}

	return nil
}

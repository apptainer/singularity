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
	b *types.Bundle
}

// Assemble creates a Sandbox image from a Bundle
func (a *SandboxAssembler) Assemble(b *types.Bundle, path string) (err error) {

	a.b = b
	defer os.RemoveAll(b.Path)

	//insert help
	err = a.insertHelpScript()
	if err != nil {
		return
	}

	//append environment
	err = a.appendEnvScript()
	if err != nil {
		return
	}

	//insert runscript
	err = a.insertRunScript()
	if err != nil {
		return
	}

	//insert test script
	err = a.insertTestScript()
	if err != nil {
		return
	}

	//move bundle rootfs to sandboxdir as final sandbox
	sylog.Debugf("Moving sandbox from %v to %v", b.Rootfs(), path)
	if err := os.Rename(b.Rootfs(), path); err != nil {
		sylog.Errorf("Sandbox Assemble Failed", err.Error())
		return err
	}

	return nil
}

func (a *SandboxAssembler) insertHelpScript() (err error) {
	//this becomes .singularity.d/runscript.help
	return nil
}

func (a *SandboxAssembler) appendEnvScript() (err error) {
	//this goes onto .singularity.d/env/90-environment.sh
	return nil
}

func (a *SandboxAssembler) insertRunScript() (err error) {
	//this becomes .singularity.d/runscript
	return nil
}

func (a *SandboxAssembler) insertTestScript() (err error) {
	//this becomes .singularity.d/actions/test
	return nil
}

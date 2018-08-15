// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package assemblers

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/singularityware/singularity/src/pkg/build/types"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

// SandboxAssembler doesnt store anything
type SandboxAssembler struct {
	b *types.Bundle
}

// Assemble creates a Sandbox image from a Bundle
func (a *SandboxAssembler) Assemble(b *types.Bundle, path string) (err error) {
	//Consider changing the interface so that bundles and part of assembler declaration?
	a.b = b
	defer os.RemoveAll(b.Path)

	//insert help
	err = a.insertHelpScript()
	if err != nil {
		return fmt.Errorf("While inserting help script: %v", err)
	}

	//insert labels
	err = a.insertLabelsJSON()
	if err != nil {
		return fmt.Errorf("While inserting labels JSON: %v", err)
	}

	//append environment
	err = a.appendEnvScript()
	if err != nil {
		return fmt.Errorf("While inserting environment script: %v", err)
	}

	//insert runscript
	err = a.insertRunScript()
	if err != nil {
		return fmt.Errorf("While inserting runscript: %v", err)
	}

	//insert test script
	err = a.insertTestScript()
	if err != nil {
		return fmt.Errorf("While inserting test script: %v", err)
	}

	//move bundle rootfs to sandboxdir as final sandbox
	sylog.Debugf("Moving sandbox from %v to %v", b.Rootfs(), path)
	if err := os.Rename(b.Rootfs(), path); err != nil {
		sylog.Errorf("Sandbox Assemble Failed", err.Error())
		return err
	}

	return nil
}

func (a *SandboxAssembler) insertHelpScript() error {
	err := ioutil.WriteFile(filepath.Join(a.b.Rootfs(), "/.singularity.d/runscript.help"), []byte(a.b.Recipe.ImageData.Help+"\n"), 0664)
	return err
}

func (a *SandboxAssembler) appendEnvScript() error {
	err := ioutil.WriteFile(filepath.Join(a.b.Rootfs(), "/.singularity.d/env/90-environment.sh"), []byte(a.b.Recipe.ImageData.Environment+"\n"), 0775)
	return err
}

func (a *SandboxAssembler) insertRunScript() error {
	err := ioutil.WriteFile(filepath.Join(a.b.Rootfs(), "/.singularity.d/runscript"), []byte(a.b.Recipe.ImageData.Runscript), 0775)
	return err
}

func (a *SandboxAssembler) insertTestScript() error {
	err := ioutil.WriteFile(filepath.Join(a.b.Rootfs(), "/.singularity.d/actions/test"), []byte(a.b.Recipe.ImageData.Test), 0775)
	return err
}

func (a *SandboxAssembler) insertLabelsJSON() error {

	text := "{\n"

	for key, val := range a.b.Recipe.ImageData.Labels {
		text += "    \"" + key + "\": \"" + val + "\"\n"
	}

	text += "}"

	err := ioutil.WriteFile(filepath.Join(a.b.Rootfs(), "/.singularity.d/labels.json"), []byte(text), 0664)
	return err
}

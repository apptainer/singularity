// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package assemblers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
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

	//insert help
	err = insertHelpScript(b)
	if err != nil {
		return fmt.Errorf("While inserting help script: %v", err)
	}

	//insert labels
	err = insertLabelsJSON(b)
	if err != nil {
		return fmt.Errorf("While inserting labels JSON: %v", err)
	}

	//append environment
	err = insertEnvScript(b)
	if err != nil {
		return fmt.Errorf("While inserting environment script: %v", err)
	}

	//insert runscript
	err = insertRunScript(b)
	if err != nil {
		return fmt.Errorf("While inserting runscript: %v", err)
	}

	//insert startscript
	err = insertStartScript(b)
	if err != nil {
		return fmt.Errorf("While inserting startscript: %v", err)
	}

	//insert test script
	err = insertTestScript(b)
	if err != nil {
		return fmt.Errorf("While inserting test script: %v", err)
	}

	//insert definition
	err = insertDefinition(b)
	if err != nil {
		return fmt.Errorf("While inserting definition: %v", err)
	}

	//move bundle rootfs to sandboxdir as final sandbox
	sylog.Debugf("Moving sandbox from %v to %v", b.Rootfs(), path)
	if err := os.Rename(b.Rootfs(), path); err != nil {
		sylog.Errorf("Sandbox Assemble Failed: %s", err)
		return err
	}

	return nil
}

func insertHelpScript(b *types.Bundle) error {
	err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/runscript.help"), []byte(b.Recipe.ImageData.Help+"\n"), 0664)
	return err
}

func insertEnvScript(b *types.Bundle) error {
	err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/env/90-environment.sh"), []byte("#!/bin/sh\n\n"+b.Recipe.ImageData.Environment+"\n"), 0775)
	return err
}

func insertRunScript(b *types.Bundle) error {
	err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/runscript"), []byte("#!/bin/sh\n\n"+b.Recipe.ImageData.Runscript+"\n"), 0775)
	return err
}

func insertStartScript(b *types.Bundle) error {
	err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/startscript"), []byte("#!/bin/sh\n\n"+b.Recipe.ImageData.Startscript+"\n"), 0775)
	return err
}

func insertTestScript(b *types.Bundle) error {
	err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/test"), []byte("#!/bin/sh\n\n"+b.Recipe.ImageData.Test+"\n"), 0775)
	return err
}

func insertDefinition(b *types.Bundle) error {
	f, err := os.Create(filepath.Join(b.Rootfs(), "/.singularity.d/Singularity"))
	if err != nil {
		return err
	}

	err = f.Chmod(0644)
	if err != nil {
		return err
	}

	b.Recipe.WriteDefinitionFile(f)

	return nil
}

func insertLabelsJSON(b *types.Bundle) error {

	text, err := json.Marshal(b.Recipe.ImageData.Labels)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/labels.json"), []byte(text), 0664)
	return err
}

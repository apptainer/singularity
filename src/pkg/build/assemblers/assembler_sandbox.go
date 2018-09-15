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

	sylog.Infof("Creating sandbox directory...")

	// insert help
	err = insertHelpScript(b)
	if err != nil {
		return fmt.Errorf("While inserting help script: %v", err)
	}

	// insert labels
	err = insertLabelsJSON(b)
	if err != nil {
		return fmt.Errorf("While inserting labels JSON: %v", err)
	}

	// insert definition
	err = insertDefinition(b)
	if err != nil {
		return fmt.Errorf("While inserting definition: %v", err)
	}

	// move bundle rootfs to sandboxdir as final sandbox
	sylog.Debugf("Moving sandbox from %v to %v", b.Rootfs(), path)
	if err := os.Rename(b.Rootfs(), path); err != nil {
		sylog.Errorf("Sandbox Assemble Failed: %s", err)
		return err
	}

	return nil
}

func insertHelpScript(b *types.Bundle) error {
	if b.RunSection("help") && b.Recipe.ImageData.Help != "" {
		sylog.Infof("Adding help info")
		err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/runscript.help"), []byte(b.Recipe.ImageData.Help+"\n"), 0664)
		if err != nil {
			return err
		}
	}
	return nil
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

	if b.RunSection("labels") && len(b.Recipe.ImageData.Labels) > 0 {
		sylog.Infof("Adding labels")
		text, err := json.Marshal(b.Recipe.ImageData.Labels)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/labels.json"), []byte(text), 0664)
		if err != nil {
			return err
		}
	}
	return nil
}

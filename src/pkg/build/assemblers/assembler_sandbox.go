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
	"strconv"

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
	if _, err := os.Stat(path); err == nil {
		os.RemoveAll(path)
	}
	if err := os.Rename(b.Rootfs(), path); err != nil {
		sylog.Errorf("Sandbox Assemble Failed: %s", err)
		return err
	}

	return nil
}

func insertHelpScript(b *types.Bundle) error {
	if b.RunSection("help") && b.Recipe.ImageData.Help != "" {
		_, err := os.Stat(filepath.Join(b.Rootfs(), "/.singularity.d/runscript.help"))
		if err != nil || b.Force {
			sylog.Infof("Adding help info")
			err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/runscript.help"), []byte(b.Recipe.ImageData.Help+"\n"), 0664)
			if err != nil {
				return err
			}
		} else {
			sylog.Warningf("Help message already exists and force option is false, not overwriting")
		}
	}
	return nil
}

func insertDefinition(b *types.Bundle) error {

	// if update, check for existing definition and move it to bootstrap history
	if b.Update {
		if _, err := os.Stat(filepath.Join(b.Rootfs(), "/.singularity.d/Singularity")); err == nil {
			// make bootstrap_history directory if it doesnt exist
			if _, err := os.Stat(filepath.Join(b.Rootfs(), "/.singularity.d/bootstrap_history")); err != nil {
				err = os.Mkdir(filepath.Join(b.Rootfs(), "/.singularity.d/bootstrap_history"), 0755)
				if err != nil {
					return err
				}
			}

			// look at number of files in bootstrap_history to give correct file name
			files, err := ioutil.ReadDir(filepath.Join(b.Rootfs(), "/.singularity.d/bootstrap_history"))

			// name is "Singularity" concatinated with an index based on number of other files in bootstrap_history
			len := strconv.Itoa(len(files))

			histName := "Singularity" + len

			// move old definition into bootstrap_history
			err = os.Rename(filepath.Join(b.Rootfs(), "/.singularity.d/Singularity"), filepath.Join(b.Rootfs(), "/.singularity.d/bootstrap_history", histName))
			if err != nil {
				return err
			}
		}

	}
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

		var text []byte

		if _, err := os.Stat(filepath.Join(b.Rootfs(), "/.singularity.d/labels.json")); err == nil {
			existingLabels := make(map[string]string)
			// check for labels that already exist
			jsonFile, err := os.Open(filepath.Join(b.Rootfs(), "/.singularity.d/labels.json"))
			if err != nil {
				return err
			}
			defer jsonFile.Close()

			jsonBytes, err := ioutil.ReadAll(jsonFile)
			if err != nil {
				return err
			}

			err = json.Unmarshal(jsonBytes, &existingLabels)
			if err != nil {
				return err
			}

			// add new labels to new map and check for collisions
			for key, value := range b.Recipe.ImageData.Labels {
				if _, ok := existingLabels[key]; ok {
					// overwrite collision if force flag is set
					if b.Force {
						existingLabels[key] = value
					} else {
						sylog.Warningf("Label: %s already exists and force option is false, not overwriting", key)
					}
				}
			}

			// make new map into json
			text, err = json.Marshal(existingLabels)
			if err != nil {
				return err
			}
		} else {
			text, err = json.Marshal(b.Recipe.ImageData.Labels)
			if err != nil {
				return err
			}
		}

		err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/labels.json"), []byte(text), 0664)
		if err != nil {
			return err
		}
	}
	return nil
}

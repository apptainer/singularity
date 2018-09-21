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
	"time"

	"github.com/otiai10/copy"
	"github.com/sylabs/singularity/src/pkg/build/types"
	"github.com/sylabs/singularity/src/pkg/buildcfg"
	"github.com/sylabs/singularity/src/pkg/sylog"
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
		if err := copy.Copy(b.Rootfs(), path); err != nil {
			sylog.Errorf("Sandbox Assemble Failed: %s", err)
			return err
		}

		if err := os.RemoveAll(b.Rootfs()); err != nil {
			sylog.Errorf("Unable to remove Bundle directory: %s", err)
			return err
		}
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

func insertLabelsJSON(b *types.Bundle) (err error) {
	var text []byte
	labels := make(map[string]string)

	if err = getExistingLabels(labels, b); err != nil {
		return err
	}

	if err = addBuildLabels(labels, b); err != nil {
		return err
	}

	if b.RunSection("labels") && len(b.Recipe.ImageData.Labels) > 0 {
		sylog.Infof("Adding labels")

		// add new labels to new map and check for collisions
		for key, value := range b.Recipe.ImageData.Labels {
			if _, ok := labels[key]; ok {
				// overwrite collision if force flag is set
				if b.Force {
					labels[key] = value
				} else {
					sylog.Warningf("Label: %s already exists and force option is false, not overwriting", key)
				}
			}
		}
	}

	// make new map into json
	text, err = json.MarshalIndent(labels, "", "\t")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/labels.json"), []byte(text), 0664)
	return err
}

func getExistingLabels(labels map[string]string, b *types.Bundle) error {
	// check for existing labels in bundle
	if _, err := os.Stat(filepath.Join(b.Rootfs(), "/.singularity.d/labels.json")); err == nil {

		jsonFile, err := os.Open(filepath.Join(b.Rootfs(), "/.singularity.d/labels.json"))
		if err != nil {
			return err
		}
		defer jsonFile.Close()

		jsonBytes, err := ioutil.ReadAll(jsonFile)
		if err != nil {
			return err
		}

		err = json.Unmarshal(jsonBytes, &labels)
		if err != nil {
			return err
		}
	}
	return nil
}

func addBuildLabels(labels map[string]string, b *types.Bundle) error {
	// schema version
	labels["org.label-schema.schema-version"] = "1.0"

	// build date and time, lots of time formatting
	currentTime := time.Now()
	year, month, day := currentTime.Date()
	date := strconv.Itoa(day) + `_` + month.String() + `_` + strconv.Itoa(year)
	hour, min, sec := currentTime.Clock()
	time := strconv.Itoa(hour) + `:` + strconv.Itoa(min) + `:` + strconv.Itoa(sec)
	zone, _ := currentTime.Zone()
	timeString := currentTime.Weekday().String() + `_` + date + `_` + time + `_` + zone
	labels["org.label-schema.build-date"] = timeString

	// singularity version
	labels["org.label-schema.usage.singularity.version"] = buildcfg.PACKAGE_VERSION

	// help info if help exists in the definition and is run in the build
	if b.RunSection("help") && b.Recipe.ImageData.Help != "" {
		labels["org.label-schema.usage"] = "/.singularity.d/runscript.help"
		labels["org.label-schema.usage.singularity.runscript.help"] = "/.singularity.d/runscript.help"
	}

	// bootstrap header info, only if this build actually bootstrapped
	if !b.Update || b.Force {
		for key, value := range b.Recipe.Header {
			labels["org.label-schema.usage.singularity.deffile."+key] = value
		}
	}

	return nil
}

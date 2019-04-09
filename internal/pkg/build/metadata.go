// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
)

func (s *stage) insertMetadata() (err error) {
	// insert help
	err = insertHelpScript(s.b)
	if err != nil {
		return fmt.Errorf("While inserting help script: %v", err)
	}

	// insert labels
	err = insertLabelsJSON(s.b)
	if err != nil {
		return fmt.Errorf("While inserting labels JSON: %v", err)
	}

	// insert definition
	err = insertDefinition(s.b)
	if err != nil {
		return fmt.Errorf("While inserting definition: %v", err)
	}

	// insert environment
	err = insertEnvScript(s.b)
	if err != nil {
		return fmt.Errorf("While inserting environment script: %v", err)
	}

	// insert startscript
	err = insertStartScript(s.b)
	if err != nil {
		return fmt.Errorf("While inserting startscript: %v", err)
	}

	// insert runscript
	err = insertRunScript(s.b)
	if err != nil {
		return fmt.Errorf("While inserting runscript: %v", err)
	}

	// insert test script
	err = insertTestScript(s.b)
	if err != nil {
		return fmt.Errorf("While inserting test script: %v", err)
	}

	return
}

func insertEnvScript(b *types.Bundle) error {
	if b.RunSection("environment") && b.Recipe.ImageData.Environment.Script != "" {
		sylog.Infof("Adding environment to container")
		envScriptPath := filepath.Join(b.Rootfs(), "/.singularity.d/env/90-environment.sh")
		_, err := os.Stat(envScriptPath)
		if os.IsNotExist(err) {
			err := ioutil.WriteFile(envScriptPath, []byte("#!/bin/sh\n\n"+b.Recipe.ImageData.Environment.Script+"\n"), 0755)
			if err != nil {
				return err
			}
		} else {
			// append to script if it already exists
			f, err := os.OpenFile(envScriptPath, os.O_APPEND|os.O_WRONLY, 0755)
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = f.WriteString("\n" + b.Recipe.ImageData.Environment.Script + "\n")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// runscript and starscript should use this function to properly handle args and shebangs
func handleShebangScript(s types.Script) (string, string) {
	shebang := "#!/bin/sh"
	script := ""
	if strings.HasPrefix(strings.TrimSpace(s.Script), "#!") {
		// separate and cleanup shebang
		split := strings.SplitN(s.Script, "\n", 2)
		shebang = strings.TrimSpace(split[0])
		if len(split) == 2 {
			script = split[1]
		}
	} else {
		script = s.Script
	}

	if s.Args != "" {
		// add arg after trimming comments
		shebang += " " + strings.Split(s.Args, "#")[0]
	}
	return shebang, script
}

func insertRunScript(b *types.Bundle) error {
	if b.RunSection("runscript") && b.Recipe.ImageData.Runscript.Script != "" {
		sylog.Infof("Adding runscript")
		shebang, script := handleShebangScript(b.Recipe.ImageData.Runscript)
		err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/runscript"), []byte(shebang+"\n\n"+script+"\n"), 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertStartScript(b *types.Bundle) error {
	if b.RunSection("startscript") && b.Recipe.ImageData.Startscript.Script != "" {
		sylog.Infof("Adding startscript")
		shebang, script := handleShebangScript(b.Recipe.ImageData.Startscript)
		err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/startscript"), []byte(shebang+"\n\n"+script+"\n"), 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertTestScript(b *types.Bundle) error {
	if b.RunSection("test") && b.Recipe.ImageData.Test.Script != "" {
		sylog.Infof("Adding testscript")
		err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/test"), []byte("#!/bin/sh\n\n"+b.Recipe.ImageData.Test.Script+"\n"), 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertHelpScript(b *types.Bundle) error {
	if b.RunSection("help") && b.Recipe.ImageData.Help.Script != "" {
		_, err := os.Stat(filepath.Join(b.Rootfs(), "/.singularity.d/runscript.help"))
		if err != nil || b.Opts.Force {
			sylog.Infof("Adding help info")
			err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/runscript.help"), []byte(b.Recipe.ImageData.Help.Script+"\n"), 0644)
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
	if b.Opts.Update {
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
			if err != nil {
				return err
			}

			// name is "Singularity" concatenated with an index based on number of other files in bootstrap_history
			len := strconv.Itoa(len(files))
			histName := "Singularity" + len
			// move old definition into bootstrap_history
			err = os.Rename(filepath.Join(b.Rootfs(), "/.singularity.d/Singularity"), filepath.Join(b.Rootfs(), "/.singularity.d/bootstrap_history", histName))
			if err != nil {
				return err
			}
		}

	}

	err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/Singularity"), b.Recipe.Raw, 0644)
	if err != nil {
		return err
	}

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
			// check if label already exists
			if _, ok := labels[key]; ok {
				// overwrite collision if it exists and force flag is set
				if b.Opts.Force {
					labels[key] = value
				} else {
					sylog.Warningf("Label: %s already exists and force option is false, not overwriting", key)
				}
			} else {
				// set if it doesnt
				labels[key] = value
			}
		}
	}

	// make new map into json
	text, err = json.MarshalIndent(labels, "", "\t")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/labels.json"), []byte(text), 0644)
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
	if b.RunSection("help") && b.Recipe.ImageData.Help.Script != "" {
		labels["org.label-schema.usage"] = "/.singularity.d/runscript.help"
		labels["org.label-schema.usage.singularity.runscript.help"] = "/.singularity.d/runscript.help"
	}

	// bootstrap header info, only if this build actually bootstrapped
	if !b.Opts.Update || b.Opts.Force {
		for key, value := range b.Recipe.Header {
			labels["org.label-schema.usage.singularity.deffile."+key] = value
		}
	}

	return nil
}

// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
)

func (s *stage) insertScripts() error {
	// insert helpfile
	if err := insertHelpScript(s.b); err != nil {
		return fmt.Errorf("while inserting help script: %v", err)
	}

	// insert definition
	if err := insertDefinition(s.b); err != nil {
		return fmt.Errorf("while inserting definition: %v", err)
	}

	// insert environment
	if err := insertEnvScript(s.b); err != nil {
		return fmt.Errorf("while inserting environment script: %v", err)
	}

	// insert startscript
	if err := insertStartScript(s.b); err != nil {
		return fmt.Errorf("while inserting startscript: %v", err)
	}

	// insert runscript
	if err := insertRunScript(s.b); err != nil {
		return fmt.Errorf("while inserting runscript: %v", err)
	}

	// insert test script
	if err := insertTestScript(s.b); err != nil {
		return fmt.Errorf("while inserting test script: %v", err)
	}

	return nil
}

func insertEnvScript(b *types.Bundle) error {
	if b.RunSection("environment") && b.Recipe.ImageData.Environment.Script != "" {
		sylog.Infof("Adding environment to container")
		envScriptPath := filepath.Join(b.RootfsPath, "/.singularity.d/env/90-environment.sh")
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
		err := ioutil.WriteFile(filepath.Join(b.RootfsPath, "/.singularity.d/runscript"), []byte(shebang+"\n\n"+script+"\n"), 0755)
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
		err := ioutil.WriteFile(filepath.Join(b.RootfsPath, "/.singularity.d/startscript"), []byte(shebang+"\n\n"+script+"\n"), 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertTestScript(b *types.Bundle) error {
	if b.RunSection("test") && b.Recipe.ImageData.Test.Script != "" {
		sylog.Infof("Adding testscript")
		err := ioutil.WriteFile(filepath.Join(b.RootfsPath, "/.singularity.d/test"), []byte("#!/bin/sh\n\n"+b.Recipe.ImageData.Test.Script+"\n"), 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertHelpScript(b *types.Bundle) error {
	if b.RunSection("help") && b.Recipe.ImageData.Help.Script != "" {
		_, err := os.Stat(filepath.Join(b.RootfsPath, "/.singularity.d/runscript.help"))
		if err != nil || b.Opts.Force {
			sylog.Infof("Adding help info")
			err := ioutil.WriteFile(filepath.Join(b.RootfsPath, "/.singularity.d/runscript.help"), []byte(b.Recipe.ImageData.Help.Script+"\n"), 0644)
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
		if _, err := os.Stat(filepath.Join(b.RootfsPath, "/.singularity.d/Singularity")); err == nil {
			// make bootstrap_history directory if it doesnt exist
			if _, err := os.Stat(filepath.Join(b.RootfsPath, "/.singularity.d/bootstrap_history")); err != nil {
				err = os.Mkdir(filepath.Join(b.RootfsPath, "/.singularity.d/bootstrap_history"), 0755)
				if err != nil {
					return err
				}
			}

			// look at number of files in bootstrap_history to give correct file name
			files, err := ioutil.ReadDir(filepath.Join(b.RootfsPath, "/.singularity.d/bootstrap_history"))
			if err != nil {
				return err
			}

			// name is "Singularity" concatenated with an index based on number of other files in bootstrap_history
			len := strconv.Itoa(len(files))
			histName := "Singularity" + len
			// move old definition into bootstrap_history
			err = os.Rename(filepath.Join(b.RootfsPath, "/.singularity.d/Singularity"), filepath.Join(b.RootfsPath, "/.singularity.d/bootstrap_history", histName))
			if err != nil {
				return err
			}
		}

	}

	err := ioutil.WriteFile(filepath.Join(b.RootfsPath, "/.singularity.d/Singularity"), b.Recipe.Raw, 0644)
	if err != nil {
		return err
	}

	return nil
}

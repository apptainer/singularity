// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/pkg/sylog"
)

const goMod = `module %s

go 1.13

require github.com/sylabs/singularity v0.0.0

replace github.com/sylabs/singularity => ./singularity_source
`

const mainGo = `package main

import (
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

// Plugin is the only variable which a plugin MUST export.
// This symbol is accessed by the plugin framework to initialize the plugin
var Plugin = pluginapi.Plugin{
	Manifest: pluginapi.Manifest{
		Name:        "%s",
		Author:      "Put your name or mail here",
		Version:     "0.1.0",
		Description: "Put a nice description",
	},
	Callbacks: []pluginapi.Callback{},
	Install:   installCallback,
}

func installCallback(path string) error {
	// Create required stuff during "plugin install"
	// (eg: configuration file, setup ...). Be careful
	// during setup as this callback is executed with
	// root privileges.
	return nil
}

// Write plugin callbacks here and register them in Callbacks
`

const gitIgnore = `singularity_source
*.sif
*.o
*.a
`

// Create creates a skeleton plugin directory structure
// to start development of a new plugin.
func Create(path, name string) error {
	dir, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("could not determine absolute path for %s: %s", path, err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("while creating plugin directory %s: %s", dir, err)
	}

	// create go.mod skeleton
	filename := filepath.Join(dir, "go.mod")
	content := fmt.Sprintf(goMod, name)
	if err := ioutil.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("while creating plugin %s: %s", filename, err)
	}

	// create main.go skeleton
	filename = filepath.Join(dir, "main.go")
	content = fmt.Sprintf(mainGo, name)
	if err := ioutil.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("while creating plugin %s: %s", filename, err)
	}

	// create .gitignore skeleton
	filename = filepath.Join(dir, ".gitignore")
	if err := ioutil.WriteFile(filename, []byte(gitIgnore), 0644); err != nil {
		return fmt.Errorf("while creating plugin %s: %s", filename, err)
	}

	// create symlink to singularity source directory
	source := filepath.Join(dir, SingularitySource)

	if _, err := os.Stat(buildcfg.SOURCEDIR); os.IsNotExist(err) {
		ls := fmt.Sprintf("ln -s /path/to/singularity/source %s", source)
		sylog.Warningf("Singularity source %s doesn't exist, you would have to execute manually %q", buildcfg.SOURCEDIR, ls)
		return nil
	} else if err != nil {
		return fmt.Errorf("while getting %s information: %s", source, err)
	}

	if err := os.Symlink(buildcfg.SOURCEDIR, source); err != nil {
		return fmt.Errorf("while creating symlink %s: %s", source, err)
	}

	return nil
}

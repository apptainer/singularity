// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"fmt"
	"path/filepath"
	"plugin"

	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

// initialized stores whether or not the plugin system has been initialized. A
// call to Initialize MUST be made before any other functions can be called.
var initialized = false

func assertInitialized() {
	if !initialized {
		panic("Plugin system has not been initialized")
	}
}

var loadedPlugins []*pluginapi.Plugin

// InitializeAll loads all plugins into memory and stores their symbols
func InitializeAll(glob string) error {
	paths, err := filepath.Glob(glob)
	if err != nil {
		return fmt.Errorf("while globbing %s: %s", glob, err)
	}

	for _, path := range paths {
		if _, err := Initialize(path); err != nil {
			return fmt.Errorf("while initializing %s as plugin: %s", path, err)
		}
	}

	initialized = true // set initialized to true
	return nil
}

// Initialize loads the plugin located at path and returns it
func Initialize(path string) (*pluginapi.Plugin, error) {
	pl, err := open(path)
	if err != nil {
		return nil, err
	}

	loadedPlugins = append(loadedPlugins, pl)
	pl.Init()

	return pl, nil
}

func open(path string) (*pluginapi.Plugin, error) {
	pluginPointer, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}

	pluginObject, err := getPluginObject(pluginPointer)
	if err != nil {
		return nil, err
	}

	return pluginObject, nil
}

func getPluginObject(pl *plugin.Plugin) (*pluginapi.Plugin, error) {
	sym, err := pl.Lookup(pluginapi.PluginSymbol)
	if err != nil {
		return nil, err
	}

	p, ok := sym.(*pluginapi.Plugin)
	if !ok {
		return nil, fmt.Errorf("symbol \"Plugin\" not of type Plugin")
	}

	return p, nil

}

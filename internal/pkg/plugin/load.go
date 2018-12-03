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

var loadedPlugins []*pluginapi.Plugin

// Initialize loads all plugins into memory and stores their symbols
func Initialize(path string) error {
	paths, err := filepath.Glob(filepath.Join(path, "*"))
	if err != nil {
		return err
	}

	for _, path := range paths {
		pl, err := open(path)
		if err != nil {
			return err
		}

		loadedPlugins = append(loadedPlugins, pl)
		pl.Init()
	}

	return nil
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

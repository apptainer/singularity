// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package syplugin

import (
	"fmt"
	"sync"

	"github.com/sylabs/singularity/internal/pkg/build/types"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

var registeredBuildPlugins BuildPluginRegistry

func init() {
	registeredBuildPlugins = BuildPluginRegistry{
		Plugins: make(map[string]BuildPlugin),
	}
}

// BasePluginRegistry ...
type BasePluginRegistry struct {
	sync.Mutex
}

// BuildPluginRegistry ...
type BuildPluginRegistry struct {
	BasePluginRegistry
	Plugins map[string]BuildPlugin
}

// RegisterBuildPlugin adds the plugin to the known plugins
func RegisterBuildPlugin(_pl interface{}) error {
	pl, ok := _pl.(BuildPlugin)
	if !ok {
		return nil
	}

	registeredBuildPlugins.Lock()
	defer registeredBuildPlugins.Unlock()

	if _, ok := registeredBuildPlugins.Plugins[pl.Name()]; ok {
		return fmt.Errorf("plugin name already registered: %s", pl.Name())
	}

	registeredBuildPlugins.Plugins[pl.Name()] = pl
	return nil
}

// GetBuildPlugins returns the list of known plugins
func GetBuildPlugins() map[string]BuildPlugin {
	registeredBuildPlugins.Lock()
	defer registeredBuildPlugins.Unlock()

	return registeredBuildPlugins.Plugins
}

// BuildHandleSections runs the HandleSection() hook on every plugin
func BuildHandleSections(i, s string) {
	var plwait sync.WaitGroup

	for name, pl := range GetBuildPlugins() {
		plwait.Add(1)
		go func(name string, pl BuildPlugin) {
			defer plwait.Done()
			sylog.Debugf("Running %s plugin: HandleSection() hook", name)

			pl.HandleSection(i, s)
		}(name, pl)
	}

	plwait.Wait()
}

// BuildHandleBundles runs the HandleBundle() hook on every plugin
func BuildHandleBundles(b *types.Bundle) {
	var plwait sync.WaitGroup

	for name, pl := range GetBuildPlugins() {
		plwait.Add(1)
		go func(name string, pl BuildPlugin) {
			defer plwait.Done()
			sylog.Debugf("Running %s plugin: HandleBundle() hook", name)

			pl.HandleBundle(b)
		}(name, pl)
	}

	plwait.Wait()
}

// BuildHandlePosts runs the HandleBundle() hook on every plugin
func BuildHandlePosts() (ret string) {
	for name, pl := range GetBuildPlugins() {
		sylog.Debugf("Running %s plugin: HandlePost() hook", name)

		ret += pl.HandlePost()
	}

	return
}

// BuildPlugin is the interface for plugins on the build system
type BuildPlugin interface {
	Name() string
	HandleSection(string, string)
	HandleBundle(*types.Bundle)
	HandlePost() string
}

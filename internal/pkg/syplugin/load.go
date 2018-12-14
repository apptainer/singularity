// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package syplugin

import (
	"fmt"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/src/plugins/apps"
)

type pluginRegisterFn func(interface{}) error

var pluginRegisterFuncs = map[string]pluginRegisterFn{
	"BuildPlugin": RegisterBuildPlugin,
}

func loadPlugins(pattern string) (pls []*plugin.Plugin, err error) {
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	for _, path := range paths {
		pl, err := plugin.Open(path)
		if err != nil {
			return nil, err
		}

		pls = append(pls, pl)
	}

	return pls, nil
}

func initPlugin(_pl *plugin.Plugin) error {
	_new, err := _pl.Lookup("New")
	if err != nil {
		return err
	}

	new, ok := _new.(func() interface{})
	if !ok {
		return fmt.Errorf("Unable to get plugin new symbol")
	}
	pl := new()
	registerPlugin(pl)

	return nil
}

func registerPlugin(pl interface{}) {
	var regWait sync.WaitGroup

	for plType, regFn := range pluginRegisterFuncs {
		regWait.Add(1)
		go func(plType string, regFn pluginRegisterFn) {
			sylog.Debugf("Registering plugin as type %s", plType)

			if err := regFn(pl); err != nil {
				sylog.Fatalf("Unable to register plugin: %s", err)
			}
			regWait.Done()
		}(plType, regFn)
	}

	regWait.Wait()
}

// InitDynamic initializes plugins via dynamic loading. This is implemented but not
// fully featured, so we're using a static methodology until 3.1
func InitDynamic() {
	var plLoadWait sync.WaitGroup
	pls, err := loadPlugins(filepath.Join(buildcfg.LIBDIR, "singularity/plugin/*"))
	if err != nil {
		sylog.Fatalf("Unable to load plugins from dir: %s", err)
	}

	for _, pl := range pls {
		plLoadWait.Add(1)
		go func(pl *plugin.Plugin) {
			defer plLoadWait.Done()
			if err := initPlugin(pl); err != nil {
				sylog.Fatalf("Something went wrong: %s", err)
			}
		}(pl)
	}

	plLoadWait.Wait()
}

// Init initializes plugins via static linking
func Init() {
	registerPlugin(apps.New())
}

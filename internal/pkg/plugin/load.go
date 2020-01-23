// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"errors"
	"fmt"
	"plugin"
	"strings"
	"sync"

	callback "github.com/sylabs/singularity/internal/pkg/plugin/callback"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

type loadedPlugins struct {
	metas   []*Meta
	plugins map[string]struct{}
	sync.Mutex
}

var lp loadedPlugins

// LoadCallbacks loads plugins registered for the hook instance passed in parameter.
func LoadCallbacks(cb pluginapi.Callback) ([]pluginapi.Callback, error) {
	callbackName := callback.Name(cb)

	if err := initMetaPlugin(); err != nil {
		return nil, err
	}

	var errs []error

	for _, meta := range lp.metas {
		if !meta.Enabled {
			continue
		}

		for _, name := range meta.Callbacks {
			if name == callbackName {
				if err := loadCallbacks(meta.binaryName()); err != nil {
					// This might be destroying information by
					// grabbing only the textual description of the
					// error
					wrappedErr := fmt.Errorf("while initializing plugin %q: %s", meta.Name, err)
					errs = append(errs, wrappedErr)
				}
			}
		}
	}

	if len(errs) > 0 {
		// Collect all the errors into a single one that can be
		// returned.
		//
		// Beware that we are destroying information that might
		// be part of the type underlying the error interface we
		// are getting here. UI-wise this might not be ideal,
		// because the user might end up seeing a bunch of
		// errors "slightly" separated by "; ".
		//
		// The alternative is to implement a type that collects
		// the individual errors and implements the error
		// interface by doing something similar to this. If
		// there's some code that needs to handle errors in a
		// more discrete way, it could type-assert an interface
		// to check if it's possible to obtain the individual
		// errors.
		var b strings.Builder
		for i, err := range errs {
			if i > 0 {
				b.WriteString("; ")
			}
			b.WriteString(err.Error())
		}
		return nil, errors.New(b.String())
	}

	return callback.Loaded(cb)
}

// initMetaPlugin reads plugin metadata files and stores data
// in the loaded plugin instance.
func initMetaPlugin() error {
	var err error

	lp.Lock()
	defer lp.Unlock()

	if lp.metas != nil {
		return nil
	}
	if lp.plugins == nil {
		lp.plugins = make(map[string]struct{})
	}

	lp.metas, err = List()
	if err != nil {
		return fmt.Errorf("while getting plugin's metadata: %s", err)
	}

	return nil
}

// loadCallbacks loads the plugin and the plugin callbacks.
func loadCallbacks(path string) error {
	lp.Lock()
	defer lp.Unlock()

	if _, ok := lp.plugins[path]; ok {
		return nil
	}

	pl, err := LoadObject(path)
	if err != nil {
		return err
	}

	lp.plugins[path] = struct{}{}

	for _, c := range pl.Callbacks {
		callback.Load(c)
	}

	return nil
}

// LoadObject loads a plugin object in memory and returns
// the Plugin object set within the plugin.
func LoadObject(path string) (*pluginapi.Plugin, error) {
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

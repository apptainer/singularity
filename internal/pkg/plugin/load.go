// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"errors"
	"fmt"
	"plugin"
	"strings"

	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

// InitializeAll loads all plugins into memory and stores their symbols.
// A call to InitializeAll MUST be made only only once.
func InitializeAll(libexecdir string) error {
	metas, err := List(libexecdir)
	if err != nil {
		return err
	}

	var errs []error

	for _, meta := range metas {
		if !meta.Enabled {
			continue
		}

		if _, err := Initialize(meta.binaryName()); err != nil {
			// This might be destroying information by
			// grabbing only the textual description of the
			// error
			wrappedErr := fmt.Errorf("while initializin plugin %q: %s", meta.Name, err)
			errs = append(errs, wrappedErr)
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
		return errors.New(b.String())
	}

	return nil
}

// Initialize loads the plugin located at path and returns it.
func Initialize(path string) (*pluginapi.Plugin, error) {
	pl, err := open(path)
	if err != nil {
		return nil, err
	}

	reg := registrar{pl.Name}
	if err := pl.Initialize(reg); err != nil {
		return nil, fmt.Errorf("could not initialize plugin: %v", err)
	}
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

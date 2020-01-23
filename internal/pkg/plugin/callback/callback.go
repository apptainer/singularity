// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package callback

import (
	"fmt"
	"unsafe"

	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

// pluginCallback contains hook callbacks function registered
// by loaded plugins.
var pluginCallbacks = make(map[string][]pluginapi.Callback)

// sameType compares if the two interfaces have the same type.
func sameType(a interface{}, b interface{}) bool {
	ptrA := unsafe.Pointer(&a)
	ptrB := unsafe.Pointer(&b)
	return *(*unsafe.Pointer)(ptrA) == *(*unsafe.Pointer)(ptrB)
}

// Loaded returns the loaded plugin callbacks of callback
// type passed in argument.
func Loaded(callbackType pluginapi.Callback) ([]pluginapi.Callback, error) {
	callbacks := pluginCallbacks[Name(callbackType)]

	// we ensure the plugin callback correspond to the registered callback
	for _, callback := range callbacks {
		if !sameType(callbackType, callback) {
			return nil, fmt.Errorf("plugin callback has type '%T' instead of '%T'", callback, callbackType)
		}
	}

	return callbacks, nil
}

// Load loads a plugin callback.
func Load(callback pluginapi.Callback) {
	name := Name(callback)

	if pluginCallbacks[name] == nil {
		pluginCallbacks[name] = make([]pluginapi.Callback, 0)
	}

	pluginCallbacks[name] = append(pluginCallbacks[name], callback)
}

// Names returns a list of unique callback name passed in argument.
func Names(callbacks []pluginapi.Callback) []string {
	var s []string

	for _, c := range callbacks {
		hookName := Name(c)
		// get rid of duplicated hook name
		found := false
		for _, name := range s {
			if name == hookName {
				found = true
				break
			}
		}
		if !found {
			s = append(s, hookName)
		}
	}

	return s
}

// Name returns the callback name.
func Name(callback pluginapi.Callback) string {
	return fmt.Sprintf("%T", callback)
}

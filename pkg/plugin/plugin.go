// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package plugin

// PluginRootDirSymbol is the name of a variable of type string which
// a plugin may define if it want to obtain the plugin root directory.
// Concatenated with the manifest name it allows to retrieve the path
// where the plugin is currently installed.
const PluginRootDirSymbol = "PluginRootDir"

// PluginSymbol is the name of a variable of type plugin.Plugin which all
// plugin implementations MUST define.
const PluginSymbol = "Plugin"

// Plugin is the "meta-type" which encompasses the plugins
// implementation through Callbacks and a Manifest
// (potentially more to be added). The plugin implementation must
// have an exported symbol named "Plugin" of this type.
//
// An example of how this will look from the plugin main package:
//
//     package main
//
//     import (
//             clicallback "github.com/sylabs/singularity/pkg/callback/cli"
//	           pluginapi "github.com/sylabs/singularity/pkg/plugin"
//     )
//
//     var Plugin = pluginapi.Plugin{
//             Manifest: pluginapi.Manifest{
//                     Name:        "PluginExample",
//                     Author:      "Sylabs Team",
//                     Version:     "v0.0.1",
//                     Description: "This is an example plugin",
//             },
//             Callbacks: []pluginapi.Callback{
//				       (clicallback.Command)(callbackRegisterCmd),
//             },
//     }
//
//     func callbackRegisterCmd(manager *cmdline.CommandManager) {
//             // Do command registration
//     }
type Plugin struct {
	// Manifest contains the plugin manifest holding
	// basic information about the plugin.
	Manifest
	// Callbacks contains plugin callbacks to be called
	// by Singularity.
	Callbacks []Callback
	// Install is a function called during singularity
	// plugin install, the function take the directory
	// where plugin object will reside and can be used
	// to store configuration files/datas needed by a
	// plugin.
	Install func(string) error
}

// Callback defines a plugin callback. Available callbacks are
// defined in pkg/plugin/callback.
type Callback interface{}

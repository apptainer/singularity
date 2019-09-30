// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"github.com/sylabs/singularity/pkg/cmdline"
	singularity "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
)

// PluginSymbol is the name of a variable of type plugin.Plugin which all
// plugin implementations MUST define.
const PluginSymbol = "Plugin"

// Plugin is the "meta-type" which encompasses the plugins
// implementation struct (Initializer interface), and a Manifest
// (potentially more to be added). The plugin implementation must
// have an exported symbol named "Plugin" of this type.
//
// An example of how this will look from a plugins main package:
//     type myPluginImplementation struct {...}
//
//     func (pl myPluginImplementation) Initialize(r Registry) {
//             // Do some initialization work!
//     }
//
//     var Plugin = syplugin.Plugin{
//             syplugin.Manifest{
//                     "MyPlugin",
//                     "Michael Bauer",
//                     "v0.0.1",
//                     "This is a test plugin",
//             },
//             myPluginImplementation{...},
//     }
type Plugin struct {
	Manifest
	Initializer
}

// Initializer is an interface which stores the object of a plugin's implementation. The Initialize
// method allows the plugin to register its functions with the Runtime to be called later.
type Initializer interface {
	Initialize(Registry) error
}

// Registry exposes functions to a plugin during Initialize()
// which allows the plugin to register its plugin hooks.
type Registry interface {
	AddCLIMutator(m CLIMutator) error
	AddEngineConfigMutator(m EngineConfigMutator) error
}

type CLIMutator struct {
	Mutate func(*cmdline.CommandManager)
}

type EngineConfigMutator struct {
	Mutate func(*singularity.EngineConfig)
}

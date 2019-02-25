// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package plugin

// PluginSymbol is the name of a variable of type plugin.Plugin which all
// plugin implementations MUST define.
const PluginSymbol = "Plugin"

// DefaultConfigSymbol is the name of a variable that a plugin can optionally
// define which stores the plugins config type initialized with default values.
// This variable will be used when installing a plugin to initialize its config
// file to the appropriate default values. If empty, default Go initialization
// values will be used.
//
//     type Config struct {
//             FeatureEnabled bool
//     }
//
//     var DefaultConfig = &Config{true}
const DefaultConfigSymbol = "DefaultConfig"

// Plugin is the "meta-type" which encompasses the plugins implementation struct (PluginType interface),
// and a type (potentially more to be added) which is defined in the syplugin package (Manifest). The plugin
// implementation must have an exported symbol named "Plugin" of type syplugin.Plugin.
//
// An example of how this will look from a plugins main package:
//     type myPluginImplementation struct {...}
//
//     func (pl myPluginImplementation) Init() {
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
//
// In addition, type Plugin also exposes useful helper methods to a plugin
type Plugin struct {
	Manifest
	Config
	Initializer
}

// Config is an empty interface which can optionally store plugin-specific configuration
// files. This can contain any fields, as long as they are able to be Marshaled/Unmarshaled
// into/from YAML format. Optionally this interface can implement the yaml.Marshaler and
// yaml.Unmarshaler interfaces for custom parsing behavior. For ease of use, Config should
// generally be assigned to a pointer to a concrete type stored inside the plugin's
// Initializer implementation.
//
// For an example of this design pattern and others, see the example plugins included
// with Singularity.
type Config interface{}

// Initializer is an interface which stores the object of a plugin's implementation. The Initialize
// method allows the plugin to register its functions with the Runtime to be called later
type Initializer interface {
	Initialize(HookRegistration)
}

// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please
// consult the LICENSE.md file distributed with the sources of this
// project regarding your rights to use or distribute this software.

// This program is used by singularity in PluginCompile.
//
// We need to get plugin's manifest in a separate to process to avoid
// the case, when we are compiling a plugin, which is already installed.
// Otherwise we will get an error `plugin already loaded` while opening
// the plugin.
package main

import (
	"encoding/json"
	"log"
	"os"
	"plugin"

	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("plugin.so path and destination are required.")
	}

	in := os.Args[1]
	out := os.Args[2]

	pluginPointer, err := plugin.Open(in)
	if err != nil {
		log.Fatal(err)
	}

	sym, err := pluginPointer.Lookup(pluginapi.PluginSymbol)
	if err != nil {
		log.Fatal(err)
	}

	p, ok := sym.(*pluginapi.Plugin)
	if !ok {
		log.Fatal(`symbol "Plugin" not of type Plugin`)
	}

	f, err := os.OpenFile(out, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(p.Manifest); err != nil {
		log.Fatal(err)
	}
}

// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
	singularity "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
)

// Plugin is the only variable which a plugin MUST export. This symbol is accessed
// by the plugin framework to initialize the plugin
var Plugin = pluginapi.Plugin{
	Manifest: pluginapi.Manifest{
		Name:        "sylabs.io/test-plugin",
		Author:      "Michael Bauer",
		Version:     "0.0.1",
		Description: "This is a short test plugin for Singularity",
	},

	Initializer: impl,
}

type pluginImplementation struct {
}

var impl = pluginImplementation{}

func (p pluginImplementation) Initialize(r pluginapi.HookRegistration) {
	// Adding a custom flag to the action commands
	flag := pluginapi.StringFlagHook{
		Flag: pflag.Flag{
			Name:      "test-flag",
			Shorthand: "z",
			Usage:     "test mc testface",
			DefValue:  "TEST_VALUE_DEFAULT",
		},
		Callback: func(f *pflag.Flag, cfg *singularity.EngineConfig) {
			fmt.Println("Calling back into plugin!")
			fmt.Printf("Received flag value: %s\n", f.Value.String())
		},
	}

	r.RegisterStringFlag(flag)

	// Adding a custom command to the root command
	cmd := pluginapi.CommandHook{
		Command: &cobra.Command{
			DisableFlagsInUseLine: true,
			Args:                  cobra.MinimumNArgs(1),
			Use:                   "test-cmd [args ...]",
			Short:                 "Test test test",
			Long:                  "Long test long test long test",
			Example:               "singularity test-cmd my test",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("test-cmd is printing args:", args)
			},
			TraverseChildren: true,
		},
	}

	r.RegisterCommand(cmd)
}

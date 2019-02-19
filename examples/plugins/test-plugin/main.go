// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package main

import (
	"fmt"

	"github.com/spf13/cobra"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

var Plugin = pluginapi.Plugin{
	Manifest: pluginapi.Manifest{
		Name:        "TestPlugin",
		Author:      "Michael Bauer",
		Version:     "0.0.1",
		Description: "This is a short test plugin for Singularity",
	},

	Initializer: Impl,
}

type PluginImplementation struct {
}

var Impl = PluginImplementation{}

func (p PluginImplementation) Init() {
}

func (p PluginImplementation) CommandAdd() []*cobra.Command {
	ret := []*cobra.Command{}

	ret = append(ret, &cobra.Command{
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
	})

	return ret
}

// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

// Plugin is the only variable which a plugin MUST export.
// This symbol is accessed by the plugin framework to initialize the plugin.
var Plugin = pluginapi.Plugin{
	Manifest: pluginapi.Manifest{
		Name:        "sylabs.io/cli-plugin",
		Author:      "Toxic Team",
		Version:     "0.0.2",
		Description: "This is a short test CLI plugin for Singularity",
	},

	Initializer: pluginImplementation{},
}

type pluginImplementation struct{}

func (p pluginImplementation) Initialize(r pluginapi.Registry) {
	r.AddCLIMutator(pluginapi.CLIMutator{
		CmdName: "version",
		Mutate: func(c *pluginapi.Cmd) {
			var test string
			c.Flags().StringVar(&test, "test", "this is a test flag from plugin", "some text to print")

			f := c.PreRun
			c.PreRun = func(c *cobra.Command, args []string) {
				fmt.Printf("test: %v\n", test)
				if f != nil {
					f(c, args)
				}
			}
		},
	})

	r.AddCLIMutator(pluginapi.CLIMutator{
		CmdName: "verify",
		Mutate: func(c *pluginapi.Cmd) {
			var abort bool
			c.Flags().BoolVar(&abort, "abort", false, "should the verify command be aborted?")

			f := c.PreRunE
			c.PreRunE = func(c *cobra.Command, args []string) error {
				if f != nil {
					if err := f(c, args); err != nil {
						return err
					}
				}

				if abort {
					return errors.New("aborting verify from the plugin")
				}
				return nil
			}
		},
	})

	r.AddCLIMutator(pluginapi.CLIMutator{
		CmdName: "break",
		Mutate: func(c *pluginapi.Cmd) {
			fmt.Println("This should not be called")
		},
	})

	r.AddCLIMutator(pluginapi.CLIMutator{
		CmdName: "singularity",
		Mutate: func(c *pluginapi.Cmd) {
			c.AddCommand(&cobra.Command{
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
		},
	})
}

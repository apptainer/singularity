// Copyright (c) 2020-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"fmt"
	"log/syslog"
	"os"

	"github.com/spf13/cobra"
	"github.com/hpcng/singularity/pkg/cmdline"
	pluginapi "github.com/hpcng/singularity/pkg/plugin"
	clicallback "github.com/hpcng/singularity/pkg/plugin/callback/cli"
	"github.com/hpcng/singularity/pkg/sylog"
)

// Plugin is the only variable which a plugin MUST export.
// This symbol is accessed by the plugin framework to initialize the plugin
var Plugin = pluginapi.Plugin{
	Manifest: pluginapi.Manifest{
		Name:        "github.com/hpcng/singularity/log-plugin",
		Author:      "Sylabs Team",
		Version:     "0.2.0",
		Description: "Log executed CLI commands to syslog",
	},
	Callbacks: []pluginapi.Callback{
		(clicallback.Command)(logCommand),
	},
}

func logCommand(manager *cmdline.CommandManager) {
	rootCmd := manager.GetRootCmd()

	// Keep track of an existing PreRunE so we can call it
	f := rootCmd.PersistentPreRunE

	// The log action is added as a PreRunE on the main `singularity` root command
	// so we can log anything a user does with `singularity`.
	rootCmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		uid := os.Getuid()
		gid := os.Getgid()
		command := c.Name()
		msg := fmt.Sprintf("UID=%d GID=%d COMMAND=%s ARGS=%v", uid, gid, command, args)

		// This logger never errors, only warns, if it fails to write to syslog
		w, err := syslog.New(syslog.LOG_INFO, "singularity")
		if err != nil {
			sylog.Warningf("Could not create syslog: %v", err)
		} else {
			defer w.Close()
			if err := w.Info(msg); err != nil {
				sylog.Warningf("Could not write to syslog: %v", err)
			}
		}

		// Call any existing PreRunE
		if f != nil {
			return f(c, args)
		}

		return nil
	}
}

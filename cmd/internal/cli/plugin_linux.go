// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
)

func init() {
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginUninstallCmd)
	// pluginCmd.AddCommand(pluginEnableCmd)
	// pluginCmd.AddCommand(pluginDisableCmd)
	pluginCmd.AddCommand(pluginCompileCmd)

	SingularityCmd.AddCommand(pluginCmd)
}

// pluginCmd is the root command for all plugin related functionalities
// which are exposed via the CLI.
//
// singularity plugin [...]
var pluginCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Invalid command")
	},
	DisableFlagsInUseLine: true,

	Use:           docs.PluginUse,
	Short:         docs.PluginShort,
	Long:          docs.PluginLong,
	Example:       docs.PluginExample,
	Aliases:       []string{"plugins"},
	SilenceErrors: true,
}

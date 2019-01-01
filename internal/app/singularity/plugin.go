// Copyright (c) 2018, Sylabs Inc. All rights reserved.
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
	// === For Future Use ===
	// PluginCmd.AddCommand(PluginListCommand)
	// PluginCmd.AddCommand(PluginInstallCommand)
	// PluginCmd.AddCommand(PluginUninstallCommand)
	// PluginCmd.AddCommand(PluginEnableCommand)
	// PluginCmd.AddCommand(PluginDisableCommand)
	PluginCmd.AddCommand(PluginCompileCmd)

	SingularityCmd.AddCommand(PluginCmd)
}

// PluginCmd is the root command for all plugin related functionalities
// which are exposed via the CLI.
//
// singularity plugin [...]
var PluginCmd = &cobra.Command{
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

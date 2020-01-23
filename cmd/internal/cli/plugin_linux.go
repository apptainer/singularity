// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/pkg/cmdline"
)

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterCmd(PluginCmd)
		cmdManager.RegisterSubCmd(PluginCmd, PluginListCmd)
		cmdManager.RegisterSubCmd(PluginCmd, PluginInstallCmd)
		cmdManager.RegisterSubCmd(PluginCmd, PluginUninstallCmd)
		cmdManager.RegisterSubCmd(PluginCmd, PluginEnableCmd)
		cmdManager.RegisterSubCmd(PluginCmd, PluginDisableCmd)
		cmdManager.RegisterSubCmd(PluginCmd, PluginCompileCmd)
		cmdManager.RegisterSubCmd(PluginCmd, PluginInspectCmd)
		cmdManager.RegisterSubCmd(PluginCmd, PluginCreateCmd)
	})
}

// PluginCmd is the root command for all plugin related functionality
// which is exposed via the CLI.
//
// singularity plugin [...]
var PluginCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("invalid command")
	},
	DisableFlagsInUseLine: true,

	Use:           docs.PluginUse,
	Short:         docs.PluginShort,
	Long:          docs.PluginLong,
	Example:       docs.PluginExample,
	Aliases:       []string{"plugins"},
	SilenceErrors: true,
}

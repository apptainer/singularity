// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func ensurePluginCmdRootPriv(cmd *cobra.Command, args []string) {
	if os.Geteuid() != 0 {
		sylog.Fatalf("command 'plugin %s' requires root privileges", cmd.Name())
	}
}

func init() {
	PluginCmd.AddCommand(PluginListCmd)
	PluginCmd.AddCommand(PluginInstallCmd)
	PluginCmd.AddCommand(PluginUninstallCmd)
	PluginCmd.AddCommand(PluginEnableCmd)
	PluginCmd.AddCommand(PluginDisableCmd)
	PluginCmd.AddCommand(PluginCompileCmd)
	PluginCmd.AddCommand(PluginInspectCmd)

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

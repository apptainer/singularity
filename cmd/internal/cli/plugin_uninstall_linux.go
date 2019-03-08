// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
)

// PluginUninstall takes the name of a plugin and uninstalls it from the
// plugin directory
//
// singularity plugin uninstall <name>
var PluginUninstallCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		err := singularity.UninstallPlugin(args[0], buildcfg.LIBEXECDIR)
		if err != nil {
			fmt.Println(err)
		}
		return err
	},
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),

	Use:     docs.PluginUninstallUse,
	Short:   docs.PluginUninstallShort,
	Long:    docs.PluginUninstallLong,
	Example: docs.PluginUninstallExample,
}

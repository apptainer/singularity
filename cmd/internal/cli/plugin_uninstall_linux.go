// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// PluginUninstallCmd takes the name of a plugin and uninstalls it from the
// plugin directory
//
// singularity plugin uninstall <name>
var PluginUninstallCmd = &cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {
		err := singularity.UninstallPlugin(args[0], buildcfg.SYSCONFDIR, buildcfg.LIBEXECDIR)
		if err != nil {
			sylog.Fatalf("Failed to uninstall plugin %q: %s.", args[0], err)
		}
	},
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),

	Use:     docs.PluginUninstallUse,
	Short:   docs.PluginUninstallShort,
	Long:    docs.PluginUninstallLong,
	Example: docs.PluginUninstallExample,
}

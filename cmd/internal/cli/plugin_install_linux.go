// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/pkg/sylog"
)

// PluginInstallCmd takes a compiled plugin.sif file and installs it
// in the appropriate location.
//
// singularity plugin install <path>
var PluginInstallCmd = &cobra.Command{
	PreRun: CheckRootOrUnpriv,
	Run: func(cmd *cobra.Command, args []string) {
		err := singularity.InstallPlugin(args[0])
		if err != nil {
			sylog.Fatalf("Failed to install plugin %q: %s.", args[0], err)
		}
	},
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),

	Use:     docs.PluginInstallUse,
	Short:   docs.PluginInstallShort,
	Long:    docs.PluginInstallLong,
	Example: docs.PluginInstallExample,
}

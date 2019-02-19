// Copyright (c) 2018, Sylabs Inc. All rights reserved.
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

// PluginInstallCmd takes a compiled plugin.sif file and installs it
// in the appropriate location
//
// singularity plugin install <path>
var PluginInstallCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Installing plugin")

		return singularity.InstallPlugin(args[0], buildcfg.LIBEXECDIR)
	},
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),

	Use:     docs.PluginInstallUse,
	Short:   docs.PluginInstallShort,
	Long:    docs.PluginInstallLong,
	Example: docs.PluginInstallExample,
}

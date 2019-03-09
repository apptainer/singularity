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

var (
	pluginName string
)

func init() {
	pluginInstallCmd.Flags().StringVarP(&pluginName, "name", "n", "", "Name to install the plugin as, defaults to the value in the manifest")
}

// pluginInstallCmd takes a compiled plugin.sif file and installs it
// in the appropriate location
//
// singularity plugin install <path> [-n name]
var pluginInstallCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		err := singularity.InstallPlugin(args[0], buildcfg.LIBEXECDIR)
		if err != nil {
			fmt.Println(err)
		}
		return err
	},
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),

	Use:     docs.PluginInstallUse,
	Short:   docs.PluginInstallShort,
	Long:    docs.PluginInstallLong,
	Example: docs.PluginInstallExample,
}

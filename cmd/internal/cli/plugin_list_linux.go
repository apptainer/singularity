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

// pluginListCmd lists the plugins installed in the system
var pluginListCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		err := singularity.ListPlugins(buildcfg.LIBEXECDIR)
		if err != nil {
			fmt.Println(err)
		}
		return err
	},
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(0),

	Use:     docs.PluginListUse,
	Short:   docs.PluginListShort,
	Long:    docs.PluginListLong,
	Example: docs.PluginListExample,
}

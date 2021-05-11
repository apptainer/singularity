// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/hpcng/singularity/docs"
	"github.com/hpcng/singularity/internal/app/singularity"
	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/spf13/cobra"
)

// PluginListCmd lists the plugins installed in the system.
var PluginListCmd = &cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {
		err := singularity.ListPlugins()
		if err != nil {
			sylog.Fatalf("Failed to get a list of installed plugins: %s.", err)
		}
	},
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(0),

	Use:     docs.PluginListUse,
	Short:   docs.PluginListShort,
	Long:    docs.PluginListLong,
	Example: docs.PluginListExample,
}

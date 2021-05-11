// Copyright (c) 2020, Sylabs Inc. All rights reserved.
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

// PluginCreateCmd creates a plugin skeleton directory
// structure to start developing a new plugin.
//
// singularity plugin create <directory> <name>
var PluginCreateCmd = &cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {
		name := args[1]
		dir := args[0]

		err := singularity.CreatePlugin(dir, name)
		if err != nil {
			sylog.Fatalf("Failed to create plugin directory %s: %s.", dir, err)
		}
	},
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(2),

	Use:     docs.PluginCreateUse,
	Short:   docs.PluginCreateShort,
	Long:    docs.PluginCreateLong,
	Example: docs.PluginCreateExample,
}

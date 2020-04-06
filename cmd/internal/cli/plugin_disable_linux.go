// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/pkg/sylog"
)

// PluginDisableCmd disables the named plugin.
//
// singularity plugin disable <name>
var PluginDisableCmd = &cobra.Command{
	PreRun: CheckRootOrUnpriv,
	Run: func(cmd *cobra.Command, args []string) {
		err := singularity.DisablePlugin(args[0], buildcfg.LIBEXECDIR)
		if err != nil {
			if os.IsNotExist(err) {
				sylog.Fatalf("Failed to disable plugin %q: plugin not found.", args[0])
			}

			// The above call to sylog.Fatalf terminates the
			// program, so we are either printing the above
			// or this, not both.
			sylog.Fatalf("Failed to disable plugin %q: %s.", args[0], err)
		}
	},
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),

	Use:     docs.PluginDisableUse,
	Short:   docs.PluginDisableShort,
	Long:    docs.PluginDisableLong,
	Example: docs.PluginDisableExample,
}

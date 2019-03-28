// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func init() {
	CapabilityAvailCmd.Flags().SetInterspersed(false)
}

// CapabilityAvailCmd singularity capability avail
var CapabilityAvailCmd = &cobra.Command{
	Args:                  cobra.RangeArgs(0, 1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		caps := ""
		if len(args) > 0 {
			caps = args[0]
		}
		c := singularity.CapAvailConfig{
			Caps: caps,
			Desc: len(args) == 0,
		}
		if err := singularity.CapabilityAvail(c); err != nil {
			sylog.Fatalf("Unable to list available capabilities: %s", err)
		}
	},

	Use:     docs.CapabilityAvailUse,
	Short:   docs.CapabilityAvailShort,
	Long:    docs.CapabilityAvailLong,
	Example: docs.CapabilityAvailExample,
}

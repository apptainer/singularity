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

func init() {
	CapabilityListCmd.Flags().SetInterspersed(false)
}

// CapabilityListCmd singularity capability list
var CapabilityListCmd = &cobra.Command{
	Args:                  cobra.RangeArgs(0, 1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		userGroup := ""
		if len(args) == 1 {
			userGroup = args[0]
		}
		c := singularity.CapListConfig{
			User:  userGroup,
			Group: userGroup,
			All:   len(args) == 0,
		}

		if err := singularity.CapabilityList(buildcfg.CAPABILITY_FILE, c); err != nil {
			sylog.Fatalf("Unable to list capabilities: %s", err)
		}
	},

	Use:     docs.CapabilityListUse,
	Short:   docs.CapabilityListShort,
	Long:    docs.CapabilityListLong,
	Example: docs.CapabilityListExample,
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"

	"github.com/singularityware/singularity/src/docs"
	"github.com/spf13/cobra"
)

func init() {
	capabilityDropCmds := []*cobra.Command{
		CapabilityDropCmd,
		// capabilityDotDropCmd,
	}

	for _, cmd := range capabilityDropCmds {
		cmd.Flags().SetInterspersed(false)
	}

	// SingularityCmd.AddCommand(capabilityDotDropCmd)
}

// CapabilityDropCmd singularity capability drop
var CapabilityDropCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(2),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("dropping capability")
	},

	Use:     docs.CapabilityDropUse,
	Short:   docs.CapabilityDropShort,
	Long:    docs.CapabilityDropLong,
	Example: docs.CapabilityDropExample,
}

/*
var capabilityDotDropCmd = &cobra.Command{
	Use:  "capability.drop [options...] <capabilities>",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("dropping capability")
	},
	Example: capabilityDropExample,
	Hidden:  true,
}
*/

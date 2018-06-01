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
	capabilityAddCmds := []*cobra.Command{
		CapabilityAddCmd,
		// capabilityDotAddCmd,
	}

	for _, cmd := range capabilityAddCmds {
		cmd.Flags().SetInterspersed(false)
	}

	// SingularityCmd.AddCommand(capabilityDotAddCmd)
}

// CapabilityAddCmd singularity capability add
var CapabilityAddCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(2),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("adding capability")
	},

	Use:     docs.CapabilityAddUse,
	Short:   docs.CapabilityAddShort,
	Long:    docs.CapabilityAddLong,
	Example: docs.CapabilityAddExample,
}

/* I'd like to have a discussion about dropping the dot version of command
groups.  Don't really want to support duplicate code that doesn't serve a real
purpose and if we are going to drop them major release is the time.
var capabilityDotAddCmd = &cobra.Command{
	Use:  "capability.add [options...] <capabilities>",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("adding capability")
	},
	Example: capabilityAddExamples,
	Hidden:  true,
}
*/

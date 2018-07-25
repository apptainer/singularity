// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"

	"github.com/singularityware/singularity/src/docs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	capabilityListCmds := []*cobra.Command{
		CapabilityListCmd,
		// capabilityDotListCmd,
	}

	var capabilityListFlags = pflag.NewFlagSet("CapabilityListFlags", pflag.ExitOnError)

	// -u|--user
	capabilityListFlags.StringVarP(&CapUser, "user", "u", "", "List capabilities for the given user")
	capabilityListFlags.SetAnnotation("user", "argtag", []string{"<user>"})

	// -g|--group
	capabilityListFlags.StringVarP(&CapGroup, "group", "g", "", "List capabilities for the given group")
	capabilityListFlags.SetAnnotation("group", "argtag", []string{"<group>"})

	// -d|--defined
	capabilityListFlags.BoolVarP(&CapDefined, "defined", "d", false, "List users and groups for those capabilities are defined")

	for _, cmd := range capabilityListCmds {
		cmd.Flags().AddFlag(capabilityListFlags.Lookup("user"))
		cmd.Flags().AddFlag(capabilityListFlags.Lookup("group"))
		cmd.Flags().AddFlag(capabilityListFlags.Lookup("defined"))
		cmd.Flags().SetInterspersed(false)
	}

	// SingularityCmd.AddCommand(capabilityDotListCmd)
}

// CapabilityListCmd singularity capability list
var CapabilityListCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(2),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("listing capability")
	},

	Use:     docs.CapabilityListUse,
	Short:   docs.CapabilityListShort,
	Long:    docs.CapabilityListLong,
	Example: docs.CapabilityListExample,
}

/*
var capabilityDotListCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("listping capability")
	},

    Use: capabilityListUse,
    Short: capabilityListShort,
    Long: capabilityListLong,
	Example: capabilityListExamples,
}
*/

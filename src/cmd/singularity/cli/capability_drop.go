// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/singularityware/singularity/src/docs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	capabilityDropCmds := []*cobra.Command{
		CapabilityDropCmd,
		// capabilityDotDropCmd,
	}

	var capabilityDropFlags = pflag.NewFlagSet("CapabilityDropFlags", pflag.ExitOnError)

	// -u|--user
	capabilityDropFlags.StringVarP(&CapUser, "user", "u", "", "Drop capabilities for the given user")
	capabilityDropFlags.SetAnnotation("user", "argtag", []string{"<user>"})

	// -g|--group
	capabilityDropFlags.StringVarP(&CapGroup, "group", "g", "", "Drop capabilities for the given group")
	capabilityDropFlags.SetAnnotation("group", "argtag", []string{"<group>"})

	// -d|--desc
	capabilityDropFlags.BoolVarP(&CapDesc, "desc", "d", false, "Print capabilities description")

	for _, cmd := range capabilityDropCmds {
		cmd.Flags().AddFlag(capabilityDropFlags.Lookup("user"))
		cmd.Flags().AddFlag(capabilityDropFlags.Lookup("group"))
		cmd.Flags().AddFlag(capabilityDropFlags.Lookup("desc"))
		cmd.Flags().SetInterspersed(false)
	}

	// SingularityCmd.AddCommand(capabilityDotDropCmd)
}

// CapabilityDropCmd singularity capability drop
var CapabilityDropCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		manageCap(args[0], capDrop)
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

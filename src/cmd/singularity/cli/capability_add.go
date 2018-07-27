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
	capabilityAddCmds := []*cobra.Command{
		CapabilityAddCmd,
		// capabilityDotAddCmd,
	}

	var capabilityAddFlags = pflag.NewFlagSet("CapabilityAddFlags", pflag.ExitOnError)

	// -u|--user
	capabilityAddFlags.StringVarP(&CapUser, "user", "u", "", "Add capabilities for the given user")
	capabilityAddFlags.SetAnnotation("user", "argtag", []string{"<user>"})

	// -g|--group
	capabilityAddFlags.StringVarP(&CapGroup, "group", "g", "", "Add capabilities for the given group")
	capabilityAddFlags.SetAnnotation("group", "argtag", []string{"<group>"})

	// -d|--desc
	capabilityAddFlags.BoolVarP(&CapDesc, "desc", "d", false, "Print capabilities description")

	for _, cmd := range capabilityAddCmds {
		cmd.Flags().AddFlag(capabilityAddFlags.Lookup("user"))
		cmd.Flags().AddFlag(capabilityAddFlags.Lookup("group"))
		cmd.Flags().AddFlag(capabilityAddFlags.Lookup("desc"))
		cmd.Flags().SetInterspersed(false)
	}

	// SingularityCmd.AddCommand(capabilityDotAddCmd)
}

// CapabilityAddCmd singularity capability add
var CapabilityAddCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		manageCap(args[0], capAdd)
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

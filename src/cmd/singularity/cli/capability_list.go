// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/singularityware/singularity/src/docs"
	"github.com/spf13/cobra"
)

func init() {

	// -u|--user
	CapabilityListCmd.Flags().StringVarP(&CapUser, "user", "u", "", "List capabilities for the given user")
	CapabilityListCmd.Flags().SetAnnotation("user", "argtag", []string{"<user>"})

	// -g|--group
	CapabilityListCmd.Flags().StringVarP(&CapGroup, "group", "g", "", "List capabilities for the given group")
	CapabilityListCmd.Flags().SetAnnotation("group", "argtag", []string{"<group>"})

	// -a|--all
	CapabilityListCmd.Flags().BoolVarP(&CapListAll, "all", "a", false, "List all users and groups capabilities")

	CapabilityListCmd.Flags().SetInterspersed(false)
}

// CapabilityListCmd singularity capability list
var CapabilityListCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(0),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		manageCap("", capList)
	},

	Use:     docs.CapabilityListUse,
	Short:   docs.CapabilityListShort,
	Long:    docs.CapabilityListLong,
	Example: docs.CapabilityListExample,
}

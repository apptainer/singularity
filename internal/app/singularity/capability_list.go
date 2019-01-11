// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build singularity_runtime

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
)

func init() {

	// -u|--user
	CapabilityListCmd.Flags().StringVarP(&CapUser, "user", "u", "", "list capabilities for the given user")
	CapabilityListCmd.Flags().SetAnnotation("user", "argtag", []string{"<user>"})
	CapabilityListCmd.Flags().SetAnnotation("user", "envkey", []string{"USER"})

	// -g|--group
	CapabilityListCmd.Flags().StringVarP(&CapGroup, "group", "g", "", "list capabilities for the given group")
	CapabilityListCmd.Flags().SetAnnotation("group", "argtag", []string{"<group>"})
	CapabilityListCmd.Flags().SetAnnotation("group", "envkey", []string{"GROUP"})

	// -a|--all
	CapabilityListCmd.Flags().BoolVarP(&CapListAll, "all", "a", false, "list all users and groups capabilities")
	CapabilityListCmd.Flags().SetAnnotation("all", "envkey", []string{"ALL"})

	CapabilityListCmd.Flags().SetInterspersed(false)
}

// CapabilityListCmd singularity capability list
var CapabilityListCmd = &cobra.Command{
	Args:                  cobra.MinimumNArgs(0),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		manageCap("", capList)
	},

	Use:     docs.CapabilityListUse,
	Short:   docs.CapabilityListShort,
	Long:    docs.CapabilityListLong,
	Example: docs.CapabilityListExample,
}

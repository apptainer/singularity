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
	CapabilityDropCmd.Flags().StringVarP(&CapUser, "user", "u", "", "Drop capabilities for the given user")
	CapabilityDropCmd.Flags().SetAnnotation("user", "argtag", []string{"<user>"})

	// -g|--group
	CapabilityDropCmd.Flags().StringVarP(&CapGroup, "group", "g", "", "Drop capabilities for the given group")
	CapabilityDropCmd.Flags().SetAnnotation("group", "argtag", []string{"<group>"})

	// -d|--desc
	CapabilityDropCmd.Flags().BoolVarP(&CapDesc, "desc", "d", false, "Print capabilities description")

	CapabilityDropCmd.Flags().SetInterspersed(false)
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

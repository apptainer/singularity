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
	CapabilityAddCmd.Flags().StringVarP(&CapUser, "user", "u", "", "Add capabilities for the given user")
	CapabilityAddCmd.Flags().SetAnnotation("user", "argtag", []string{"<user>"})

	// -g|--group
	CapabilityAddCmd.Flags().StringVarP(&CapGroup, "group", "g", "", "Add capabilities for the given group")
	CapabilityAddCmd.Flags().SetAnnotation("group", "argtag", []string{"<group>"})

	// -d|--desc
	CapabilityAddCmd.Flags().BoolVarP(&CapDesc, "desc", "d", false, "Print capabilities description")

	CapabilityAddCmd.Flags().SetInterspersed(false)
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

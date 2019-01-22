// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
)

func init() {

	// -u|--user
	CapabilityDropCmd.Flags().StringVarP(&CapUser, "user", "u", "", "drop capabilities for the given user")
	CapabilityDropCmd.Flags().SetAnnotation("user", "argtag", []string{"<user>"})
	CapabilityDropCmd.Flags().SetAnnotation("user", "envkey", []string{"USER"})

	// -g|--group
	CapabilityDropCmd.Flags().StringVarP(&CapGroup, "group", "g", "", "drop capabilities for the given group")
	CapabilityDropCmd.Flags().SetAnnotation("group", "argtag", []string{"<group>"})
	CapabilityDropCmd.Flags().SetAnnotation("group", "envkey", []string{"GROUP"})

	// -d|--desc
	CapabilityDropCmd.Flags().BoolVarP(&CapDesc, "desc", "d", false, "print capabilities description")
	CapabilityDropCmd.Flags().SetAnnotation("desc", "envkey", []string{"DESC"})

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

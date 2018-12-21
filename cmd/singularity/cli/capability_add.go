// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build linux

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/src/docs"
)

func init() {

	// -u|--user
	CapabilityAddCmd.Flags().StringVarP(&CapUser, "user", "u", "", "add capabilities for the given user")
	CapabilityAddCmd.Flags().SetAnnotation("user", "argtag", []string{"<user>"})
	CapabilityAddCmd.Flags().SetAnnotation("user", "envkey", []string{"USER"})

	// -g|--group
	CapabilityAddCmd.Flags().StringVarP(&CapGroup, "group", "g", "", "add capabilities for the given group")
	CapabilityAddCmd.Flags().SetAnnotation("group", "argtag", []string{"<group>"})
	CapabilityAddCmd.Flags().SetAnnotation("group", "envkey", []string{"GROUP"})

	// -d|--desc
	CapabilityAddCmd.Flags().BoolVarP(&CapDesc, "desc", "d", false, "print capabilities description")
	CapabilityAddCmd.Flags().SetAnnotation("desc", "envkey", []string{"DESC"})

	CapabilityAddCmd.Flags().SetInterspersed(false)
}

// CapabilityAddCmd singularity capability add
var CapabilityAddCmd = &cobra.Command{
	Args:                  cobra.MinimumNArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		manageCap(args[0], capAdd)
	},

	Use:     docs.CapabilityAddUse,
	Short:   docs.CapabilityAddShort,
	Long:    docs.CapabilityAddLong,
	Example: docs.CapabilityAddExample,
}

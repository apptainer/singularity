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

	var capabilityAddFlags = pflag.NewFlagSet("CapabilityAddFlags", pflag.ExitOnError)

	// -u|--user
	capabilityAddFlags.StringVarP(&CapUser, "user", "u", "", "Add capabilities for the given user")
	capabilityAddFlags.SetAnnotation("user", "argtag", []string{"<user>"})

	// -g|--group
	capabilityAddFlags.StringVarP(&CapGroup, "group", "g", "", "Add capabilities for the given group")
	capabilityAddFlags.SetAnnotation("group", "argtag", []string{"<group>"})

	// -d|--desc
	capabilityAddFlags.BoolVarP(&CapDesc, "desc", "d", false, "Print capabilities description")

    CapabilityAddCmd.Flags().AddFlag(capabilityAddFlags.Lookup("user"))
    CapabilityAddCmd.Flags().AddFlag(capabilityAddFlags.Lookup("group"))
    CapabilityAddCmd.Flags().AddFlag(capabilityAddFlags.Lookup("desc"))
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

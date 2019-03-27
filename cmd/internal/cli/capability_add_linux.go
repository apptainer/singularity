// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func init() {

	// -u|--user
	CapabilityAddCmd.Flags().StringVarP(&CapUser, "user", "u", "", "add capabilities to a user")
	CapabilityAddCmd.Flags().SetAnnotation("user", "argtag", []string{"<user>"})
	CapabilityAddCmd.Flags().SetAnnotation("user", "envkey", []string{"USER"})

	// -g|--group
	CapabilityAddCmd.Flags().StringVarP(&CapGroup, "group", "g", "", "add capabilities to a group")
	CapabilityAddCmd.Flags().SetAnnotation("group", "argtag", []string{"<group>"})
	CapabilityAddCmd.Flags().SetAnnotation("group", "envkey", []string{"GROUP"})

	CapabilityAddCmd.Flags().SetInterspersed(false)
}

// CapabilityAddCmd singularity capability add
var CapabilityAddCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		c := singularity.CapManageConfig{
			Caps:  args[0],
			User:  CapUser,
			Group: CapGroup,
		}

		if err := singularity.CapabilityAdd(buildcfg.CAPABILITY_FILE, c); err != nil {
			sylog.Fatalf("Unable to add capabilities: %s", err)
		}
	},

	Use:     docs.CapabilityAddUse,
	Short:   docs.CapabilityAddShort,
	Long:    docs.CapabilityAddLong,
	Example: docs.CapabilityAddExample,
}

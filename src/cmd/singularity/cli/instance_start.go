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
	options := []string{
		"add-caps",
		"allow-setuid",
		"bind",
		"boot",
		"contain",
		"containall",
		"cleanenv",
		"dns",
		"drop-caps",
		"fakeroot",
		"home",
		"hostname",
		"keep-privs",
		"net",
		"network",
		"network-args",
		"no-home",
		"no-privs",
		"nv",
		"overlay",
		"scratch",
		"security",
		"userns",
		"uts",
		"workdir",
		"writable",
	}

	for _, opt := range options {
		InstanceStartCmd.Flags().AddFlag(actionFlags.Lookup(opt))
	}

	InstanceStartCmd.Flags().SetInterspersed(false)
}

// InstanceStartCmd singularity instance start
var InstanceStartCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(2),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		a := []string{"/.singularity.d/actions/start"}
		execStarter(cmd, args[0], a, args[1])
	},

	Use:     docs.InstanceStartUse,
	Short:   docs.InstanceStartShort,
	Long:    docs.InstanceStartLong,
	Example: docs.InstanceStartExample,
}

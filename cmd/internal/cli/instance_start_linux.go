// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
)

func init() {
	options := []string{
		"add-caps",
		"allow-setuid",
		"apply-cgroups",
		"bind",
		"boot",
		"contain",
		"containall",
		"containlibs",
		"cleanenv",
		"docker-login",
		"docker-username",
		"docker-password",
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
		"no-nv",
		"no-privs",
		"nv",
		"overlay",
		"scratch",
		"security",
		"userns",
		"uts",
		"workdir",
		"writable",
		"writable-tmpfs",
	}

	for _, opt := range options {
		InstanceStartCmd.Flags().AddFlag(actionFlags.Lookup(opt))
	}

	InstanceStartCmd.Flags().SetInterspersed(false)
}

// InstanceStartCmd singularity instance start
var InstanceStartCmd = &cobra.Command{
	Args:                  cobra.MinimumNArgs(2),
	PreRun:                actionPreRun,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/start"}, args[2:]...)
		setVM(cmd)
		if VM {
			execVM(cmd, args[0], a)
			return
		}
		execStarter(cmd, args[0], a, args[1])
	},

	Use:     docs.InstanceStartUse,
	Short:   docs.InstanceStartShort,
	Long:    docs.InstanceStartLong,
	Example: docs.InstanceStartExample,
}

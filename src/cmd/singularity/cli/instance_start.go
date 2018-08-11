// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"

	"github.com/singularityware/singularity/src/docs"
	"github.com/spf13/cobra"
)

func init() {

    options := [16]string {
        "add-caps",
        "allow-setuid",
        "bind",
        "boot",
        "drop-caps",
        "fakeroot",
        "home",
        "hostname",
        "keep-privs",
        "net",
        "no-privs",
        "overlay",
        "scratch",
        "userns",
        "uts",
        "workdir",
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
		fmt.Println("starting instance")
	},

	Use:     docs.InstanceStartUse,
	Short:   docs.InstanceStartShort,
	Long:    docs.InstanceStartLong,
	Example: docs.InstanceStartExample,
}

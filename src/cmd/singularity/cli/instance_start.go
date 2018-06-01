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

	instanceStartCmds := []*cobra.Command{
		InstanceStartCmd,
		// instanceDotStartCmd,
	}

	for _, cmd := range instanceStartCmds {
		cmd.Flags().SetInterspersed(false)

		cmd.Flags().AddFlag(actionFlags.Lookup("bind"))
		cmd.Flags().AddFlag(actionFlags.Lookup("home"))
		cmd.Flags().AddFlag(actionFlags.Lookup("net"))
		cmd.Flags().AddFlag(actionFlags.Lookup("uts"))
		cmd.Flags().AddFlag(actionFlags.Lookup("overlay"))
		cmd.Flags().AddFlag(actionFlags.Lookup("scratch"))
		cmd.Flags().AddFlag(actionFlags.Lookup("workdir"))
		cmd.Flags().AddFlag(actionFlags.Lookup("userns"))
		cmd.Flags().AddFlag(actionFlags.Lookup("hostname"))
		cmd.Flags().AddFlag(actionFlags.Lookup("boot"))
		cmd.Flags().AddFlag(actionFlags.Lookup("fakeroot"))
		cmd.Flags().AddFlag(actionFlags.Lookup("keep-privs"))
		cmd.Flags().AddFlag(actionFlags.Lookup("no-privs"))
		cmd.Flags().AddFlag(actionFlags.Lookup("add-caps"))
		cmd.Flags().AddFlag(actionFlags.Lookup("drop-caps"))
		cmd.Flags().AddFlag(actionFlags.Lookup("allow-setuid"))
	}

	SingularityCmd.AddCommand(InstanceStartCmd)
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

/*
var instanceDotStartCmd = &cobra.Command{
	Use:  "instance.start [options...] <container path> <instance name>",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("starting instance")
	},
	Example: instanceStartExample,
	Hidden:  true,
}
*/

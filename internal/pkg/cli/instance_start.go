/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"fmt"

	sflags "github.com/singularityware/singularity/internal/pkg/cli/flags"
	"github.com/spf13/cobra"
)

var instanceStartExamples string = `
      $ singularity instance.start /tmp/my-sql.img mysql

      $ singularity shell instance://mysql
      Singularity my-sql.img> pwd
      /home/mibauer/mysql
      Singularity my-sql.img> ps
      PID TTY          TIME CMD
        1 pts/0    00:00:00 sinit
        2 pts/0    00:00:00 bash
        3 pts/0    00:00:00 ps
      Singularity my-sql.img> 
    
      $ singularity instance.stop /tmp/my-sql.img mysql
      Stopping /tmp/my-sql.img mysql`

func init() {
	instanceStartCmds := []*cobra.Command{
		instanceStartCmd,
		instanceDotStartCmd,
	}

	for _, cmd := range instanceStartCmds {
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("bind"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("home"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("net"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("uts"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("overlay"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("scratch"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("workdir"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("userns"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("hostname"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("boot"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("fakeroot"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("keep-privs"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("no-privs"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("add-caps"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("drop-caps"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("allow-setuid"))
	}

	singularityCmd.AddCommand(instanceDotStartCmd)
}

var instanceStartCmd = &cobra.Command{
	Use:  "start [start options...] <container path> <instance name>",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("starting instance")
	},
	Example: instanceStartExamples,
}

var instanceDotStartCmd = &cobra.Command{
	Use:  "instance.start [options...] <container path> <instance name>",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("starting instance")
	},
	Example: instanceStartExamples,
	Hidden:  true,
}

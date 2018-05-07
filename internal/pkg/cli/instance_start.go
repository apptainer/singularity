/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"fmt"

	"github.com/singularityware/singularity/docs"
	"github.com/spf13/cobra"
)

var instanceStartUse string = `start [start options...] <container path> <instance name>`

var instanceStartShort string = `
start a named instance of the given container image`

var instanceStartLong string = `
The instance start command allows you to create a new named instance from an 
existing container image that will begin running in the background. If a 
start.sh script is defined in the container metadata the commands in that
script will be executed with the instance start command as well.

singularity instance start accepts the following container formats` + formats

var instanceStartExample string = `
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

	manHelp := func(c *cobra.Command, args []string) {
		docs.DispManPg("singularity-instance-start")
	}

	instanceStartCmds := []*cobra.Command{
		InstanceStartCmd,
		// instanceDotStartCmd,
	}

	for _, cmd := range instanceStartCmds {
		cmd.Flags().SetInterspersed(false)
		cmd.SetHelpFunc(manHelp)

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

	// SingularityCmd.AddCommand(instanceDotStartCmd)
}

var InstanceStartCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(2),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("starting instance")
	},

	Use:     instanceStartUse,
	Short:   instanceStartShort,
	Long:    instanceStartLong,
	Example: instanceStartExample,
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

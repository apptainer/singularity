/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
    "github.com/singularityware/singularity/docs"
)

var instanceStopUse string = `stop [stop options...] [instance]`

var instanceStopShort string = `
stop a named instance of a given container image`

var instanceStopLong string = `
The command singularity instance stop allows you to stop and clean up a named, 
running instance of a given container image.`

var instanceStopExample string = `
$ singularity instance.start my-sql.img mysql1
$ singularity instance.start my-sql.img mysql2
$ singularity instance.stop mysql*
Stopping mysql1 instance of my-sql.img (PID=23845)
Stopping mysql2 instance of my-sql.img (PID=23858)

$ singularity instance.start my-sql.img mysql1

Force instance to shutdown
$ singularity instance.stop -f mysql1 (may corrupt data)

Send SIGTERM to the instance
$ singularity instance.stop -s SIGTERM mysql1
$ singularity instance.stop -s TERM mysql1
$ singularity instance.stop -s 15 mysql1`

func init() {

    manHelp := func(c *cobra.Command, args []string) {
        docs.DispManPg("singularity-capability-list")
    }

	// SingularityCmd.AddCommand(instanceDotStopCmd)
    InstanceStopCmd.Flags().SetInterspersed(false)
    InstanceStopCmd.SetHelpFunc(manHelp)
}

var InstanceStopCmd = &cobra.Command{
    DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("stopping instance")
	},

	Use: instanceStopUse,
    Short: instanceStopShort,
    Long: instanceStopLong,
    Example: instanceStopExample,
}

/*
var instanceDotStopCmd = &cobra.Command{
	Use:    "instance.stop",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("stopping instance")
	},
}
*/

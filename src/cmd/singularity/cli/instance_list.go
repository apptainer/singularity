/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"fmt"

	// "github.com/singularityware/singularity/docs"
	"github.com/spf13/cobra"
)

var User string

var instanceListUse string = `list [list options...] <container>`

var instanceListShort string = `
list all running and named Singularity instances`

var instanceListLong string = `
The instance list command allows you to view the Singularity container
instances that are currently running in the background.`

var instanceListExample string = `
$ singularity instance.list
DAEMON NAME      PID      CONTAINER IMAGE
test            11963     /home/mibauer/singularity/sinstance/test.img

$ sudo singularity instance.list -u mibauer
DAEMON NAME      PID      CONTAINER IMAGE
test            11963     /home/mibauer/singularity/sinstance/test.img
test2           16219     /home/mibauer/singularity/sinstance/test.img
`

func init() {
	InstanceListCmd.Flags().SetInterspersed(false)

	// SingularityCmd.AddCommand(instanceDotListCmd)
	InstanceListCmd.Flags().StringVarP(&User, "user", "u", "", `If running as root, list instances from "username">`)
}

var InstanceListCmd = &cobra.Command{
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("listing instances")
	},
	DisableFlagsInUseLine: true,

	Use:     instanceListUse,
	Short:   instanceListShort,
	Long:    instanceListLong,
	Example: instanceListExample,
}

/*
var instanceDotListCmd = &cobra.Command{
	Use:  "instance.list [list options...] [patterns]",
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("listing instances")
	},
	Hidden:                true,
	DisableFlagsInUseLine: true,
}
*/

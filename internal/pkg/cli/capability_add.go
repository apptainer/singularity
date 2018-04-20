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

var capabilityAddUse string = `add [add options...] <capabilities>`

var capabilityAddShort string = `
add Linux capabilities to a container at runtime`

var capabilityAddLong string = `
The capability add command allows you to grant fine grained Linux capabilities 
to your container at runtime. For instance, `

var capabilityAddExample string = `
$ singularity capability.add /tmp/my-sql.img mysql

$ singularity shell capability://mysql
Singularity my-sql.img> pwd
/home/mibauer/mysql
Singularity my-sql.img> ps
PID TTY          TIME CMD
1 pts/0    00:00:00 sinit
2 pts/0    00:00:00 bash
3 pts/0    00:00:00 ps
Singularity my-sql.img> 

$ singularity capability.stop /tmp/my-sql.img mysql
Stopping /tmp/my-sql.img mysql`

func init() {

    manHelp := func(c *cobra.Command, args []string) {
        docs.DispManPg("singularity-capability-add")
    }

	capabilityAddCmds := []*cobra.Command{
		CapabilityAddCmd,
		// capabilityDotAddCmd,
	}

	for _, cmd := range capabilityAddCmds {
		cmd.Flags().SetInterspersed(false)
        cmd.SetHelpFunc(manHelp)
	}

	// SingularityCmd.AddCommand(capabilityDotAddCmd)
}

var CapabilityAddCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(2),
    DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("adding capability")
	},

    Use: capabilityAddUse,
    Short: capabilityAddShort,
    Long: capabilityAddLong,
	Example: capabilityAddExample,
}

/* I'd like to have a discussion about dropping the dot version of command
groups.  Don't really want to support duplicate code that doesn't serve a real
purpose and if we are going to drop them major release is the time. 
var capabilityDotAddCmd = &cobra.Command{
	Use:  "capability.add [options...] <capabilities>",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("adding capability")
	},
	Example: capabilityAddExamples,
	Hidden:  true,
}
*/

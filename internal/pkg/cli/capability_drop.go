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
var capabilityDropUse string = `drop [drop options...] <capabilities>`

var capabilityDropShort string = `
remove Linux capabilities from your container at runtime`

var capabilityDropLong string = `
The capability drop command allows you to remove Linux capabilities from your 
container with fine grained precision. This way you can ensure that your 
container is as secure as it can be given the functions it must carry out. For 
instance, `

var capabilityDropExample string = `
$ singularity capability.drop /tmp/my-sql.img mysql

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

	capabilityDropCmds := []*cobra.Command{
		CapabilityDropCmd,
		// capabilityDotDropCmd,
	}

	for _, cmd := range capabilityDropCmds {
		cmd.Flags().SetInterspersed(false)
        cmd.SetHelpFunc(manHelp)
	}

	// SingularityCmd.AddCommand(capabilityDotDropCmd)
}

var CapabilityDropCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(2),
    DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("dropping capability")
	},

    Use: capabilityDropUse,
    Short: capabilityDropShort,
    Long: capabilityDropLong,
	Example: capabilityDropExample,
}

/*
var capabilityDotDropCmd = &cobra.Command{
	Use:  "capability.drop [options...] <capabilities>",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("dropping capability")
	},
	Example: capabilityDropExample,
	Hidden:  true,
}
*/

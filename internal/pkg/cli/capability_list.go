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
)

var capabilityListExamples string = `
      $ singularity capability.list /tmp/my-sql.img mysql

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
	capabilityListCmds := []*cobra.Command{
		CapabilityListCmd,
		capabilityDotListCmd,
	}

	for _, cmd := range capabilityListCmds {
		cmd.Flags().SetInterspersed(false)
	}

	SingularityCmd.AddCommand(capabilityDotListCmd)
}

var CapabilityListCmd = &cobra.Command{
	Use:  "list [list options...] <capabilities>",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("listping capability")
	},
	Example: capabilityListExamples,
}

var capabilityDotListCmd = &cobra.Command{
	Use:  "capability.list [options...] <capabilities>",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("listping capability")
	},
	Example: capabilityListExamples,
	Hidden:  true,
}

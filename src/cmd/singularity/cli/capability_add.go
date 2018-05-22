// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var capabilityAddExamples = `
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
	capabilityAddCmds := []*cobra.Command{
		capabilityAddCmd,
		capabilityDotAddCmd,
	}

	for _, cmd := range capabilityAddCmds {
		cmd.Flags().SetInterspersed(false)
	}

	singularityCmd.AddCommand(capabilityDotAddCmd)
}

var capabilityAddCmd = &cobra.Command{
	Use:  "add [add options...] <capabilities>",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("adding capability")
	},
	Example: capabilityAddExamples,
}

var capabilityDotAddCmd = &cobra.Command{
	Use:  "capability.add [options...] <capabilities>",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("adding capability")
	},
	Example: capabilityAddExamples,
	Hidden:  true,
}

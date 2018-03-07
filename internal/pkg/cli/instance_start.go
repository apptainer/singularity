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

func init() {
	singularityCmd.AddCommand(instanceDotStartCmd)
}

var instanceStartCmd = &cobra.Command{
	Use:  "start [...] <container path> <instance name>",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("starting instance")
	},
}

var instanceDotStartCmd = &cobra.Command{
	Use:    "instance.start",
	Args:   cobra.MinimumNArgs(2),
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("starting instance")
	},
}

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
	singularityCmd.AddCommand(instanceDotListCmd)
}

var instanceListCmd = &cobra.Command{
	Use:  "list [list options...] [patterns]",
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("listing instances")
	},
	DisableFlagsInUseLine: true,
}

var instanceDotListCmd = &cobra.Command{
	Use:  "instance.list [list options...] [patterns]",
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("listing instances")
	},
	Hidden:                true,
	DisableFlagsInUseLine: true,
}

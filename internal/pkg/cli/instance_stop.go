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
	SingularityCmd.AddCommand(instanceDotStopCmd)
}

var InstanceStopCmd = &cobra.Command{
	Use: "stop",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("stopping instance")
	},
}

var instanceDotStopCmd = &cobra.Command{
	Use:    "instance.stop",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("stopping instance")
	},
}

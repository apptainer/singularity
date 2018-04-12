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
	SignCmd.Flags().SetInterspersed(false)
	verifyCmd.Flags().SetInterspersed(false)

	SingularityCmd.AddCommand(SignCmd)
	SingularityCmd.AddCommand(verifyCmd)
}

var SignCmd = &cobra.Command{
	Use:  "sign <image path>",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains image path
		fmt.Printf("Signing image: %s\n", args[0])
	},
}

var verifyCmd = &cobra.Command{
	Use:  "verify <image path>",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains image path
		fmt.Printf("Verifying image: %s\n", args[0])
	},
}

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

var (
	Sandbox  bool
	Writable bool
	Force    bool
	NoTest   bool
	Section  string
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use: "build",
	Run: func(cmd *cobra.Command, args []string) {
		if silent {
			fmt.Println("Silent!")
		}

		if Sandbox {
			fmt.Println("Sandbox!")
		}
	},
	TraverseChildren: true,
}

func init() {
	buildCmd.Flags().SetInterspersed(false)
	singularityCmd.AddCommand(buildCmd)

	buildCmd.Flags().BoolVarP(&Sandbox, "sandbox", "s", false, "")
	buildCmd.Flags().StringVar(&Section, "section", "", "")
	buildCmd.Flags().BoolVarP(&Writable, "writable", "w", false, "")
	buildCmd.Flags().BoolVarP(&Force, "force", "f", false, "")
	buildCmd.Flags().BoolVarP(&NoTest, "notest", "T", false, "")
}

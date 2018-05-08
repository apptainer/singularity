/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"fmt"

	"github.com/singularityware/singularity/src/pkg/signing"
	"github.com/spf13/cobra"
	"os"

	"github.com/singularityware/singularity/docs"
)

var verifyUse string = `verify <image path>`

var verifyShort string = ``

var verifyLong string = ``

var verifyExample string = ``

func init() {
	manHelp := func(c *cobra.Command, args []string) {
		docs.DispManPg("singularity-verify")
	}

	verifyCmd.Flags().SetInterspersed(false)
	verifyCmd.SetHelpFunc(manHelp)
	SingularityCmd.AddCommand(verifyCmd)
}

var verifyCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args: cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains image path
		fmt.Printf("Verifying image: %s\n", args[0])
		if err := signing.Verify(args[0]); err != nil {
			os.Exit(2)
		}
	},

	Use:     verifyUse,
	Short:   verifyShort,
	Long:    verifyLong,
	Example: verifyExample,
}

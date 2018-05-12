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

	// "github.com/singularityware/singularity/docs"
)

var signUse string = `sign <image path>`

var signShort string = `Attach cryptographic signature to container`

var signLong string = ``

var signExample string = ``

func init() {
	SignCmd.Flags().SetInterspersed(false)
	SingularityCmd.AddCommand(SignCmd)
}

var SignCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args: cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains image path
		fmt.Printf("Signing image: %s\n", args[0])
		if err := signing.Sign(args[0]); err != nil {
			os.Exit(2)
		}
	},

	Use:     signUse,
	Short:   signShort,
	Long:    signLong,
	Example: signExample,
}

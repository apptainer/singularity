//   Copyright (c) 2018, Sylabs, Inc. All rights reserved.
//   This software is licensed under a 3-clause BSD license.  Please
//   consult LICENSE.md file distributed with the sources of this project regarding
//   your rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/signing"
	"github.com/spf13/cobra"
)

func init() {
	VerifyCmd.Flags().SetInterspersed(false)
	SingularityCmd.AddCommand(VerifyCmd)
}

// VerifyCmd singularity verify
var VerifyCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args: cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains image path
		fmt.Printf("Verifying image: %s\n", args[0])
		if err := signing.Verify(args[0]); err != nil {
			os.Exit(2)
		}
	},

	Use:     docs.VerifyUse,
	Short:   docs.VerifyShort,
	Long:    docs.VerifyLong,
	Example: docs.VerifyExample,
}

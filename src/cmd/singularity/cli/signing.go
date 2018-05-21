// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"

	"os"

	"github.com/singularityware/singularity/src/pkg/signing"
	"github.com/spf13/cobra"
)

func init() {
	signCmd.Flags().SetInterspersed(false)
	verifyCmd.Flags().SetInterspersed(false)

	singularityCmd.AddCommand(signCmd)
	singularityCmd.AddCommand(verifyCmd)
}

var signCmd = &cobra.Command{
	Use:  "sign <image path>",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains image path
		fmt.Printf("Signing image: %s\n", args[0])
		if err := signing.Sign(args[0]); err != nil {
			os.Exit(2)
		}
	},
}

var verifyCmd = &cobra.Command{
	Use:  "verify <image path>",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains image path
		fmt.Printf("Verifying image: %s\n", args[0])
		if err := signing.Verify(args[0]); err != nil {
			os.Exit(2)
		}
	},
}

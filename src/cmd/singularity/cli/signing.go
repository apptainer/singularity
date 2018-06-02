// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/signing"
	"github.com/spf13/cobra"
)

func init() {
	SignCmd.Flags().SetInterspersed(false)
	SingularityCmd.AddCommand(SignCmd)
}

// SignCmd singularity sign
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

	Use:     docs.SignUse,
	Short:   docs.SignShort,
	Long:    docs.SignLong,
	Example: docs.SignExample,
}

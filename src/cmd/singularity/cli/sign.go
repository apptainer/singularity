// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/signing"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/spf13/cobra"
)

func init() {
	SignCmd.Flags().SetInterspersed(false)
	SignCmd.Flags().StringVarP(&keyServerURL, "url", "u", defaultKeysServer, "specify the key server URL")
	SingularityCmd.AddCommand(SignCmd)
}

// SignCmd singularity sign
var SignCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:   cobra.ExactArgs(1),
	PreRun: sylabsToken,

	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains image path
		fmt.Printf("Signing image: %s\n", args[0])
		if err := doSignCmd(args[0], keyServerURL); err != nil {
			sylog.Errorf("signing container failed: %s", err)
			os.Exit(2)
		}
		fmt.Printf("Signature created and applied to %v\n", args[0])
	},

	Use:     docs.SignUse,
	Short:   docs.SignShort,
	Long:    docs.SignLong,
	Example: docs.SignExample,
}

func doSignCmd(cpath, url string) error {
	if err := signing.Sign(cpath, url, authToken); err != nil {
		return err
	}

	return nil
}

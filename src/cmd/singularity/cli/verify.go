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
	VerifyCmd.Flags().SetInterspersed(false)
	VerifyCmd.Flags().StringVarP(&keyServerURL, "url", "u", defaultKeysServer, "specify the key server URL")
	SingularityCmd.AddCommand(VerifyCmd)
}

// VerifyCmd singularity verify
var VerifyCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:   cobra.ExactArgs(1),
	PreRun: sylabsToken,

	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains image path
		fmt.Printf("Verifying image: %s\n", args[0])
		if err := doVerifyCmd(args[0], keyServerURL); err != nil {
			sylog.Errorf("verification failed: %s", err)
			os.Exit(2)
		}
	},

	Use:     docs.VerifyUse,
	Short:   docs.VerifyShort,
	Long:    docs.VerifyLong,
	Example: docs.VerifyExample,
}

func doVerifyCmd(cpath, url string) error {
	if err := signing.Verify(cpath, url, authToken); err != nil {
		return err
	}

	return nil
}

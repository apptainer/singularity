// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/pkg/sypgp"
)

var secret bool

func init() {
	KeyListCmd.Flags().SetInterspersed(false)

	KeyListCmd.Flags().BoolVarP(&secret, "secret", "s", false, "only list private keys")
	KeyListCmd.Flags().SetAnnotation("secret", "envkey", []string{"SECRET"})
}

// KeyListCmd is `singularity key list' and lists local store OpenPGP keys
var KeyListCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(0),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := doKeyListCmd(secret); err != nil {
			os.Exit(2)
		}
	},

	Use:     docs.KeyListUse,
	Short:   docs.KeyListShort,
	Long:    docs.KeyListLong,
	Example: docs.KeyListExample,
}

func doKeyListCmd(secret bool) error {
	if !secret {
		fmt.Printf("Public keys (%s):\n\n", sypgp.PublicPath())
		sypgp.PrintPubKeyring()
	} else {
		fmt.Printf("Private keys (%s):\n\n", sypgp.SecretPath())
		sypgp.PrintPrivKeyring()
	}

	return nil
}

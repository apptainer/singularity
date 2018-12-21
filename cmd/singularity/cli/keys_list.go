// Copyright (c) 2018, Sylabs Inc. All rights reserved.
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
	KeysListCmd.Flags().SetInterspersed(false)

	KeysListCmd.Flags().BoolVarP(&secret, "secret", "s", false, "list private keys instead of the default which displays public ones")
	KeysListCmd.Flags().SetAnnotation("secret", "envkey", []string{"SECRET"})
}

// KeysListCmd is `singularity keys list' and lists local store OpenPGP keys
var KeysListCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(0),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := doKeysListCmd(secret); err != nil {
			os.Exit(2)
		}
	},

	Use:     docs.KeysListUse,
	Short:   docs.KeysListShort,
	Long:    docs.KeysListLong,
	Example: docs.KeysListExample,
}

func doKeysListCmd(secret bool) error {
	if secret == false {
		fmt.Printf("Public key listing (%s):\n\n", sypgp.PublicPath())
		sypgp.PrintPubKeyring()
	} else {
		fmt.Printf("Private key listing (%s):\n\n", sypgp.SecretPath())
		sypgp.PrintPrivKeyring()
	}

	return nil
}

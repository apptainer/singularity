// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/sypgp"
	"github.com/spf13/cobra"

	"os"
)

var secret bool

func init() {
	KeysListCmd.Flags().SetInterspersed(false)
	KeysListCmd.Flags().BoolVarP(&secret, "secret", "s", false, "list private keys instead of the default which displays public ones")
}

// KeysListCmd is `singularity keys list' and lists local store PGP keys
var KeysListCmd = &cobra.Command{
	Args: cobra.RangeArgs(0, 1),
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

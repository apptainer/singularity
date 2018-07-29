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
	SyPgpListCmd.Flags().SetInterspersed(false)
	SyPgpListCmd.Flags().BoolVarP(&secret, "secret", "s", false, "list private keys instead of the default which displays public ones")
}

// SyPgpListCmd is `singularity sypgp list' and lists local store PGP keys
var SyPgpListCmd = &cobra.Command{
	Args: cobra.RangeArgs(0, 1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := doSyPgpListCmd(secret); err != nil {
			os.Exit(2)
		}
	},

	Use:     docs.SyPgpListUse,
	Short:   docs.SyPgpListShort,
	Long:    docs.SyPgpListLong,
	Example: docs.SyPgpListExample,
}

func doSyPgpListCmd(secret bool) error {
	if secret == false {
		fmt.Printf("Public key listing (%s):\n\n", sypgp.PublicPath())
		sypgp.PrintPubKeyring()
	} else {
		fmt.Printf("Private key listing (%s):\n\n", sypgp.SecretPath())
		sypgp.PrintPrivKeyring()
	}

	return nil
}

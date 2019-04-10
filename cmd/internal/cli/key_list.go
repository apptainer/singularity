// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"

	"github.com/sylabs/singularity/pkg/cmdline"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/pkg/sypgp"
)

var secret bool

// -s|--secret
var keyListSecretFlag = cmdline.Flag{
	ID:           "keyListSecretFlag",
	Value:        &secret,
	DefaultValue: false,
	Name:         "secret",
	ShortHand:    "s",
	Usage:        "list private keys instead of the default which displays public ones",
	EnvKeys:      []string{"SECRET"},
}

func init() {
	cmdManager.RegisterCmdFlag(&keyListSecretFlag, KeyListCmd)
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
		fmt.Printf("Public key listing (%s):\n\n", sypgp.PublicPath())
		sypgp.PrintPubKeyring()
	} else {
		fmt.Printf("Private key listing (%s):\n\n", sypgp.SecretPath())
		sypgp.PrintPrivKeyring()
	}

	return nil
}

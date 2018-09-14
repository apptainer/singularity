// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/sypgp"
	"github.com/spf13/cobra"
)

func init() {
	KeysNewPairCmd.Flags().SetInterspersed(false)
}

// KeysNewPairCmd is `singularity keys newpair' and generate a new OpenPGP key pair
var KeysNewPairCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(0),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		sypgp.GenKeyPair()
	},

	Use:     docs.KeysNewPairUse,
	Short:   docs.KeysNewPairShort,
	Long:    docs.KeysNewPairLong,
	Example: docs.KeysNewPairExample,
}

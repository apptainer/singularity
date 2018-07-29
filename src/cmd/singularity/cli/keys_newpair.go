// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/sypgp"
	"github.com/spf13/cobra"
)

func init() {
	SyPgpNewPairCmd.Flags().SetInterspersed(false)
}

// SyPgpNewPairCmd is `singularity sypgp newpair' and generate a new OpenPGP key pair
var SyPgpNewPairCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(0),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		sypgp.GenKeyPair()
	},

	Use:     docs.SyPgpNewPairUse,
	Short:   docs.SyPgpNewPairShort,
	Long:    docs.SyPgpNewPairLong,
	Example: docs.SyPgpNewPairExample,
}

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
	PgpNewPairCmd.Flags().SetInterspersed(false)
}

// PgpNewPairCmd is `singularity pgp newpair' and generate a new PGP key pair
var PgpNewPairCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(0),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		sypgp.GenKeyPair()
	},

	Use:     docs.PgpNewPairUse,
	Short:   docs.PgpNewPairShort,
	Long:    docs.PgpNewPairLong,
	Example: docs.PgpNewPairExample,
}

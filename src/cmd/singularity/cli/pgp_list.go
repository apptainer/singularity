// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/singularityware/singularity/src/docs"
	//	"github.com/singularityware/singularity/src/pkg/sypgp"
	"github.com/spf13/cobra"
)

func init() {
	pgpListCmds := []*cobra.Command{
		PgpListCmd,
	}

	for _, cmd := range pgpListCmds {
		cmd.Flags().SetInterspersed(false)
	}
}

// PgpListCmd is `singularity pgp list' and lists local store PGP keys
var PgpListCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(0),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		println("pgp: list")
	},

	Use:     docs.PgpListUse,
	Short:   docs.PgpListShort,
	Long:    docs.PgpListLong,
	Example: docs.PgpListExample,
}

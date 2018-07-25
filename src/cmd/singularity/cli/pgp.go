// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/singularityware/singularity/src/docs"
	"github.com/spf13/cobra"
)

func init() {
	SingularityCmd.AddCommand(PgpCmd)
	PgpCmd.AddCommand(PgpNewPairCmd)
	PgpCmd.AddCommand(PgpListCmd)
	PgpCmd.AddCommand(PgpPullCmd)
	PgpCmd.AddCommand(PgpPushCmd)
}

// PgpCmd is the 'pgp' command that allows management of key stores
var PgpCmd = &cobra.Command{
	Run: nil,
	DisableFlagsInUseLine: true,

	Use:     docs.PgpUse,
	Short:   docs.PgpShort,
	Long:    docs.PgpLong,
	Example: docs.PgpExample,
}

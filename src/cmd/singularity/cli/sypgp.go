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
	SingularityCmd.AddCommand(SyPgpCmd)
	SyPgpCmd.AddCommand(SyPgpNewPairCmd)
	SyPgpCmd.AddCommand(SyPgpListCmd)
	SyPgpCmd.AddCommand(SyPgpPullCmd)
	SyPgpCmd.AddCommand(SyPgpPushCmd)
}

// SyPgpCmd is the 'sypgp' command that allows management of key stores
var SyPgpCmd = &cobra.Command{
	Run: nil,
	DisableFlagsInUseLine: true,

	Use:     docs.SyPgpUse,
	Short:   docs.SyPgpShort,
	Long:    docs.SyPgpLong,
	Example: docs.SyPgpExample,
}

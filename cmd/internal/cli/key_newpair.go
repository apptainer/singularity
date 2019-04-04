// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
)

// KeyNewPairCmd is 'singularity key newpair' and generate a new OpenPGP key pair
var KeyNewPairCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(0),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := sypgp.GenKeyPair(keyServerURI, authToken); err != nil {
			sylog.Errorf("creating newpair failed: %v", err)
			os.Exit(2)
		}
	},

	Use:     docs.KeyNewPairUse,
	Short:   docs.KeyNewPairShort,
	Long:    docs.KeyNewPairLong,
	Example: docs.KeyNewPairExample,
}

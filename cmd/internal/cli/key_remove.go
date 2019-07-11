// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
)

// KeyRemoveCmd is `singularity key remove <fingerprint>' command
var KeyRemoveCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		keyring := sypgp.NewHandle("")
		err := keyring.RemovePubKey(args[0])
		if err != nil {
			sylog.Fatalf("Unable to remove public key: %s", err)
		}

	},

	Use:     docs.KeyRemoveUse,
	Short:   docs.KeyRemoveShort,
	Long:    docs.KeyRemoveLong,
	Example: docs.KeyRemoveExample,
}

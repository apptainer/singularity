// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2017-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/scs-key-client/client"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/remote/endpoint"
	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
)

// KeySearchCmd is 'singularity key search' and look for public keys from a key server
var KeySearchCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.TODO()

		keyClient, err := getKeyserverClientConfig(keyServerURI, endpoint.KeyserverSearchOp)
		if err != nil {
			sylog.Fatalf("Keyserver client failed: %s", err)
		}

		if err := doKeySearchCmd(ctx, args[0], keyClient); err != nil {
			sylog.Errorf("search failed: %s", err)
			os.Exit(2)
		}
	},

	Use:     docs.KeySearchUse,
	Short:   docs.KeySearchShort,
	Long:    docs.KeySearchLong,
	Example: docs.KeySearchExample,
}

func doKeySearchCmd(ctx context.Context, search string, c *client.Config) error {
	// get keyring with matching search string
	return sypgp.SearchPubkey(ctx, c, search, keySearchLongList)
}

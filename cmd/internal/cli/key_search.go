// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
)

// KeySearchCmd is 'singularity key search' and look for public keys from a key server
var KeySearchCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		handleKeyFlags(cmd)

		if err := doKeySearchCmd(args[0], keyServerURI); err != nil {
			sylog.Errorf("search failed: %s", err)
			os.Exit(2)
		}
	},

	Use:     docs.KeySearchUse,
	Short:   docs.KeySearchShort,
	Long:    docs.KeySearchLong,
	Example: docs.KeySearchExample,
}

func doKeySearchCmd(search string, url string) error {
	// get keyring with matching search string
	return sypgp.SearchPubkey(http.DefaultClient, search, url, authToken, keySearchLongList)
}

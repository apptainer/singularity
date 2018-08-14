// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/sypgp"
	"github.com/spf13/cobra"

	"os"
)

func init() {
	KeysSearchCmd.Flags().SetInterspersed(false)
	KeysSearchCmd.Flags().StringVarP(&url, "url", "u", "", "overwrite the default remote url")
}

// KeysSearchCmd is `singularity keys search' and look for public keys from a key server
var KeysSearchCmd = &cobra.Command{
	Args: cobra.RangeArgs(1, 2),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		if err := doKeysSearchCmd(args[0], url); err != nil {
			sylog.Errorf("search failed: %s", err)
			os.Exit(2)
		}
	},

	Use:     docs.KeysSearchUse,
	Short:   docs.KeysSearchShort,
	Long:    docs.KeysSearchLong,
	Example: docs.KeysSearchExample,
}

func doKeysSearchCmd(search string, url string) error {
	if url == "" {
		// lookup key management server URL from singularity.conf

		// else use default builtin
		url = defaultKeysServer
	}

	// get keyring with matching search string
	list, err := sypgp.SearchPubkey(search, url, authToken)
	if err != nil {
		return err
	}

	fmt.Println(list)

	return nil
}

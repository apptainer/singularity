// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
)

func init() {
	KeysSearchCmd.Flags().SetInterspersed(false)

	KeysSearchCmd.Flags().StringVarP(&keyServerURL, "url", "u", defaultKeysServer, "specify the key server URL")
	KeysSearchCmd.Flags().SetAnnotation("url", "envkey", []string{"URL"})
}

// KeysSearchCmd is `singularity keys search' and look for public keys from a key server
var KeysSearchCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		if err := doKeysSearchCmd(args[0], keyServerURL); err != nil {
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
	// get keyring with matching search string
	list, err := sypgp.SearchPubkey(search, url, authToken)
	if err != nil {
		return err
	}

	fmt.Println(list)

	return nil
}

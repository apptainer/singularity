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
	KeySearchCmd.Flags().SetInterspersed(false)

	KeySearchCmd.Flags().StringVarP(&keyServerURL, "url", "u", defaultKeyServer, "specify the key server URL")
	KeySearchCmd.Flags().SetAnnotation("url", "envkey", []string{"URL"})
}

// KeySearchCmd is `singularity key search' and look for public keys from a key server
var KeySearchCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		if err := doKeySearchCmd(args[0], keyServerURL); err != nil {
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
	list, err := sypgp.SearchPubkey(search, url, authToken)
	if err != nil {
		return err
	}

	fmt.Println(list)

	return nil
}

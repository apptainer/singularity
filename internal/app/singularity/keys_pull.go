// Copyright (c) 2018, Sylabs Inc. All rights reserved.
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
	KeysPullCmd.Flags().SetInterspersed(false)

	KeysPullCmd.Flags().StringVarP(&keyServerURL, "url", "u", defaultKeysServer, "specify the key server URL")
	KeysPullCmd.Flags().SetAnnotation("url", "envkey", []string{"URL"})
}

// KeysPullCmd is `singularity keys pull' and fetches public keys from a key server
var KeysPullCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		if err := doKeysPullCmd(args[0], keyServerURL); err != nil {
			sylog.Errorf("pull failed: %s", err)
			os.Exit(2)
		}
	},

	Use:     docs.KeysPullUse,
	Short:   docs.KeysPullShort,
	Long:    docs.KeysPullLong,
	Example: docs.KeysPullExample,
}

func doKeysPullCmd(fingerprint string, url string) error {
	var count int

	// get matching keyring
	el, err := sypgp.FetchPubkey(fingerprint, url, authToken)
	if err != nil {
		return err
	}

	elstore, err := sypgp.LoadPubKeyring()
	if err != nil {
		return err
	}

	// store in local cache
	fp, err := os.OpenFile(sypgp.PublicPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer fp.Close()

	for _, e := range el {
		for _, estore := range elstore {
			if e.PrimaryKey.KeyId == estore.PrimaryKey.KeyId {
				e = nil // Entity is already in key store
			}
		}
		if e != nil {
			if err = e.Serialize(fp); err != nil {
				return err
			}
			count++
		}
	}

	fmt.Printf("%v key(s) fetched and stored in local cache %s\n", count, sypgp.PublicPath())

	return nil
}

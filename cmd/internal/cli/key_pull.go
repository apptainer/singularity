// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
)

// KeyPullCmd is `singularity key pull' and fetches public keys from a key server
var KeyPullCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		handleKeyFlags(cmd)

		if err := doKeyPullCmd(args[0], keyServerURI); err != nil {
			sylog.Errorf("pull failed: %s", err)
			os.Exit(2)
		}
	},

	Use:     docs.KeyPullUse,
	Short:   docs.KeyPullShort,
	Long:    docs.KeyPullLong,
	Example: docs.KeyPullExample,
}

func doKeyPullCmd(fingerprint string, url string) error {
	var count int

	keyring := sypgp.NewHandle("")

	// get matching keyring
	el, err := sypgp.FetchPubkey(http.DefaultClient, fingerprint, url, authToken, false)
	if err != nil {
		return fmt.Errorf("unable to pull key from server: %v", err)
	}

	elstore, err := keyring.LoadPubKeyring()
	if err != nil {
		return err
	}

	// store in local cache
	fp, err := os.OpenFile(keyring.PublicPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer fp.Close()

	for _, e := range el {
		storeKey := true
		for _, estore := range elstore {
			if e.PrimaryKey.KeyId == estore.PrimaryKey.KeyId {
				storeKey = false // Entity is already in key store
				break
			}
		}
		if storeKey {
			if err = e.Serialize(fp); err != nil {
				return fmt.Errorf("unable to serialize key: %v", err)
			}
			count++
		}
	}

	fmt.Printf("%v key(s) added to keyring of trust %s\n", count, keyring.PublicPath())

	return nil
}

// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2017-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/scs-key-client/client"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/remote/endpoint"
	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
)

// KeyPullCmd is `singularity key pull' and fetches public keys from a key server
var KeyPullCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.TODO()

		keyClient, err := getKeyserverClientConfig(keyServerURI, endpoint.KeyserverPullOp)
		if err != nil {
			sylog.Fatalf("Keyserver client failed: %s", err)
		}

		if err := doKeyPullCmd(ctx, args[0], keyClient); err != nil {
			sylog.Errorf("pull failed: %s", err)
			os.Exit(2)
		}
	},

	Use:     docs.KeyPullUse,
	Short:   docs.KeyPullShort,
	Long:    docs.KeyPullLong,
	Example: docs.KeyPullExample,
}

func doKeyPullCmd(ctx context.Context, fingerprint string, c *client.Config) error {
	var count int

	keyring := sypgp.NewHandle("")

	// get matching keyring
	el, err := sypgp.FetchPubkey(ctx, c, fingerprint, false)
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

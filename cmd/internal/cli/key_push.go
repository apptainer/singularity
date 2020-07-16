// Copyright (c) 2017-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
)

// KeyPushCmd is `singularity key list' and lists local store OpenPGP keys
var KeyPushCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.TODO()

		handleKeyFlags(cmd)

		if err := doKeyPushCmd(ctx, args[0], keyServerURI); err != nil {
			sylog.Errorf("push failed: %s", err)
			os.Exit(2)
		}
	},

	Use:     docs.KeyPushUse,
	Short:   docs.KeyPushShort,
	Long:    docs.KeyPushLong,
	Example: docs.KeyPushExample,
}

func doKeyPushCmd(ctx context.Context, fingerprint string, url string) error {
	keyring := sypgp.NewHandle("")
	el, err := keyring.LoadPubKeyring()
	if err != nil {
		return err
	}
	if el == nil {
		return fmt.Errorf("no public keys in local store to choose from")
	}

	if len(fingerprint) != 16 && len(fingerprint) != 40 {
		return fmt.Errorf("please provide a keyid(16 chars) or a full fingerprint(40 chars)")
	}

	keyID, err := strconv.ParseUint(fingerprint[len(fingerprint)-16:], 16, 64)
	if err != nil {
		return fmt.Errorf("please provide a keyid(16 chars) or a full fingerprint(40 chars): %s", err)
	}

	keys := el.KeysById(keyID)
	if len(keys) != 1 {
		return fmt.Errorf("could not find the requested key")
	}
	entity := keys[0].Entity

	if err = sypgp.PushPubkey(ctx, http.DefaultClient, entity, url, authToken); err != nil {
		if e, ok := err.(*sypgp.ErrPushAccepted); ok {
			fmt.Printf("%s\n", e)
			return nil
		}
		return err
	}

	fmt.Printf("public key `%v' pushed to server successfully\n", fingerprint)

	return nil
}

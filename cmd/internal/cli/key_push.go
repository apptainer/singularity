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
	"strconv"

	"github.com/spf13/cobra"
	"github.com/sylabs/scs-key-client/client"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/remote/endpoint"
	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
)

// KeyPushCmd is `singularity key list' and lists local store OpenPGP keys
var KeyPushCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.TODO()

		co, err := getKeyserverClientOpts(keyServerURI, endpoint.KeyserverPushOp)
		if err != nil {
			sylog.Fatalf("Keyserver client failed: %s", err)
		}

		if err := doKeyPushCmd(ctx, args[0], co...); err != nil {
			sylog.Errorf("push failed: %s", err)
			os.Exit(2)
		}
	},

	Use:     docs.KeyPushUse,
	Short:   docs.KeyPushShort,
	Long:    docs.KeyPushLong,
	Example: docs.KeyPushExample,
}

func doKeyPushCmd(ctx context.Context, fingerprint string, co ...client.Option) error {
	var opts []sypgp.HandleOpt
	path := ""

	if keyGlobalPubKey {
		path = buildcfg.SINGULARITY_CONFDIR
		opts = append(opts, sypgp.GlobalHandleOpt())
	}

	keyring := sypgp.NewHandle(path, opts...)
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

	if err = sypgp.PushPubkey(ctx, entity, co...); err != nil {
		return err
	}

	fmt.Printf("public key `%v' pushed to server successfully\n", fingerprint)

	return nil
}

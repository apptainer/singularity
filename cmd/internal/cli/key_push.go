// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
)

func init() {
	KeyPushCmd.Flags().SetInterspersed(false)

	KeyPushCmd.Flags().StringVarP(&keyServerURL, "url", "u", defaultKeyServer, "specify the key server URL")
	KeyPushCmd.Flags().SetAnnotation("url", "envkey", []string{"URL"})
}

// KeyPushCmd is `singularity key list' and lists local store OpenPGP keys
var KeyPushCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		handleKeyFlags(cmd)

		if err := doKeyPushCmd(args[0], keyServerURL); err != nil {
			sylog.Errorf("push failed: %s", err)
			os.Exit(2)
		}
	},

	Use:     docs.KeyPushUse,
	Short:   docs.KeyPushShort,
	Long:    docs.KeyPushLong,
	Example: docs.KeyPushExample,
}

func doKeyPushCmd(fingerprint string, url string) error {
	el, err := sypgp.LoadPubKeyring()
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

	if err = sypgp.PushPubkey(entity, url, authToken); err != nil {
		return err
	}

	fmt.Printf("public key `%v' pushed to server successfully\n", fingerprint)

	return nil
}

func handleKeyFlags(cmd *cobra.Command) {
	// if we can load config and if default endpoint is set, use that
	// otherwise fall back on regular authtoken and URI behavior
	e, err := sylabsRemote(remoteConfig)
	if err == scs.ErrNoDefault {
		sylog.Warningf("No default remote in use, falling back to: %v", keyServerURL)
		return
	} else if err != nil {
		sylog.Fatalf("Unable to load remote configuration: %v", err)
	}

	authToken = e.Token
	if !cmd.Flags().Lookup("url").Changed {
		uri, err := e.GetServiceURI("keystore")
		if err != nil {
			sylog.Fatalf("Unable to get key service URI: %v", err)
		}
		keyServerURL = uri
	}
}

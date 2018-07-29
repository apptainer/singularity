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
	"strconv"
)

func init() {
	SyPgpPushCmd.Flags().SetInterspersed(false)
	SyPgpPushCmd.Flags().StringVarP(&url, "url", "u", "", "overwrite the default remote url")
}

// SyPgpPushCmd is `singularity sypgp list' and lists local store OpenPGP keys
var SyPgpPushCmd = &cobra.Command{
	Args: cobra.RangeArgs(0, 1),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		if err := doSyPgpPushCmd(args[0], url); err != nil {
			sylog.Errorf("push failed: %s", err)
			os.Exit(2)
		}
	},

	Use:     docs.SyPgpPushUse,
	Short:   docs.SyPgpPushShort,
	Long:    docs.SyPgpPushLong,
	Example: docs.SyPgpPushExample,
}

func doSyPgpPushCmd(fingerprint string, url string) error {
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

	if url == "" {
		// lookup key management server URL from singularity.conf
		url = "https://example.com:11371"
	}

	if err = sypgp.PushPubkey(entity, url, authToken); err != nil {
		return err
	}

	return nil
}

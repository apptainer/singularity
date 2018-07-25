// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/sypgp"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/openpgp"

	"os"
)

func init() {
	PgpPushCmd.Flags().SetInterspersed(false)
	PgpPushCmd.Flags().StringVarP(&url, "url", "u", "", "overwrite the default remote url")
}

var entity *openpgp.Entity

// PgpPushCmd is `singularity pgp list' and lists local store PGP keys
var PgpPushCmd = &cobra.Command{
	Args: cobra.RangeArgs(0, 1),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		if err := doPgpPushCmd(entity, url); err != nil {
			os.Exit(2)
		}
	},

	Use:     docs.PgpPushUse,
	Short:   docs.PgpPushShort,
	Long:    docs.PgpPushLong,
	Example: docs.PgpPushExample,
}

func doPgpPushCmd(entity *openpgp.Entity, url string) error {
	sylog.Debugf("entity: %v, url: %s, authToken: %s\n", entity, url, authToken)
	if url == "" {
		// lookup key management server URL from singularity.conf
		sypgp.PushPubkey(entity, "https://example.com:11371", authToken)
	} else {
		sypgp.PushPubkey(entity, url, authToken)
	}

	return nil
}

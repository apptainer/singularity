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

	"os"
)

var url string

func init() {
	PgpPullCmd.Flags().SetInterspersed(false)
	PgpPullCmd.Flags().StringVarP(&url, "url", "u", "", "overwrite the default remote url")
}

// PgpPullCmd is `singularity pgp pull' and fetches public keys from a key server
var PgpPullCmd = &cobra.Command{
	Args: cobra.RangeArgs(1, 2),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		if err := doPgpPullCmd(args[0], url); err != nil {
			os.Exit(2)
		}
	},

	Use:     docs.PgpPullUse,
	Short:   docs.PgpPullShort,
	Long:    docs.PgpPullLong,
	Example: docs.PgpPullExample,
}

func doPgpPullCmd(fingerprint string, url string) error {
	sylog.Debugf("fingerprint: %s, url: %s, authToken: %s\n", fingerprint, url, authToken)
	if url == "" {
		// lookup key management server URL from singularity.conf
		sypgp.FetchPubkey(fingerprint, "https://example.com:11371", authToken)
	} else {
		sypgp.FetchPubkey(fingerprint, url, authToken)
	}

	return nil
}

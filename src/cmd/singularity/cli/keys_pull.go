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
)

var url string

func init() {
	SyPgpPullCmd.Flags().SetInterspersed(false)
	SyPgpPullCmd.Flags().StringVarP(&url, "url", "u", "", "overwrite the default remote url")
}

// SyPgpPullCmd is `singularity sypgp pull' and fetches public keys from a key server
var SyPgpPullCmd = &cobra.Command{
	Args: cobra.RangeArgs(1, 2),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		if err := doSyPgpPullCmd(args[0], url); err != nil {
			sylog.Errorf("pull failed: %s", err)
			os.Exit(2)
		}
	},

	Use:     docs.SyPgpPullUse,
	Short:   docs.SyPgpPullShort,
	Long:    docs.SyPgpPullLong,
	Example: docs.SyPgpPullExample,
}

func doSyPgpPullCmd(fingerprint string, url string) error {
	var count int

	if url == "" {
		// lookup key management server URL from singularity.conf
		url = "https://example.com:11371"
	}

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

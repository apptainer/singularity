// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
)

func init() {
	KeyPullCmd.Flags().SetInterspersed(false)

	KeyPullCmd.Flags().StringVarP(&keyServerURL, "url", "u", defaultKeyServer, "specify the key server URL")
	KeyPullCmd.Flags().SetAnnotation("url", "envkey", []string{"URL"})
}

// KeyPullCmd is `singularity key pull' and fetches public keys from a key server
var KeyPullCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		// If flag did not change, look for a remote configuration, otherwise fall back to default
		if !cmd.Flags().Lookup("url").Changed {
			e, err := sylabsRemote(remoteConfig)
			if err == nil {
				uri, err := e.GetServiceURI("keystore")
				if err != nil {
					sylog.Fatalf("Unable to get key service URI: %v", err)
				}
				keyServerURL = uri
			} else if err == scs.ErrNoDefault {
				sylog.Warningf("No default remote in use, falling back to: %v", keyServerURL)
			} else {
				sylog.Fatalf("Unable to load remote configuration: %v", err)
			}
		}

		if err := doKeyPullCmd(args[0], keyServerURL); err != nil {
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

	// get matching keyring
	el, err := sypgp.FetchPubkey(fingerprint, url, authToken, false)
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
		storeKey := true
		for _, estore := range elstore {
			if e.PrimaryKey.KeyId == estore.PrimaryKey.KeyId {
				storeKey = false // Entity is already in key store
				break
			}
		}
		if storeKey {
			if err = e.Serialize(fp); err != nil {
				return err
			}
			count++
		}
	}

	fmt.Printf("%v key(s) fetched and stored in local cache %s\n", count, sypgp.PublicPath())

	return nil
}

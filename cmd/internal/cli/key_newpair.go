// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/sypgp"
)

var (

	// KeyNewPairCmd is 'singularity key newpair' and generate a new OpenPGP key pair
	KeyNewPairCmd = &cobra.Command{
		Args:                  cobra.ExactArgs(0),
		DisableFlagsInUseLine: true,
		PreRun:                sylabsToken,
		Run: func(cmd *cobra.Command, args []string) {
			keyring := sypgp.NewHandle("")
			handleKeyNewPairEndpoint()

			genOpts := sypgp.GenKeyPairOptions{
				Name:     keyNewPairName,
				Email:    keyNewPairEmail,
				Comment:  keyNewPairComment,
				Password: keyNewPairPassword,
			}

			if _, err := keyring.GenKeyPair(keyServerURI, authToken, genOpts); err != nil {
				sylog.Errorf("creating newpair failed: %v", err)
				os.Exit(2)
			}
		},

		Use:     docs.KeyNewPairUse,
		Short:   docs.KeyNewPairShort,
		Long:    docs.KeyNewPairLong,
		Example: docs.KeyNewPairExample,
	}

	keyNewPairName     string
	KeyNewPairNameFlag = &cmdline.Flag{
		ID:           "KeyNewPairNameFlag",
		Value:        &keyNewPairName,
		DefaultValue: "",
		Name:         "name",
		ShortHand:    "n",
		Usage:        "keys owner name",
	}

	keyNewPairEmail     string
	KeyNewPairEmailFlag = &cmdline.Flag{
		ID:           "KeyNewPairEmailFlag",
		Value:        &keyNewPairEmail,
		DefaultValue: "",
		Name:         "email",
		ShortHand:    "e",
		Usage:        "keys owner email",
	}

	keyNewPairComment     string
	KeyNewPairCommentFlag = &cmdline.Flag{
		ID:           "KeyNewPairCommentFlag",
		Value:        &keyNewPairComment,
		DefaultValue: "",
		Name:         "comment",
		ShortHand:    "c",
		Usage:        "keys comment",
	}

	keyNewPairPassword     *string
	KeyNewPairPasswordFlag = &cmdline.Flag{
		ID:           "KeyNewPairPasswordFlag",
		Value:        &keyNewPairPassword,
		DefaultValue: nil,
		Name:         "password",
		ShortHand:    "p",
		Usage:        "keys password",
	}
)

func handleKeyNewPairEndpoint() {
	// if we can load config and if default endpoint is set, use that
	// otherwise fall back on regular authtoken and URI behavior
	endpoint, err := sylabsRemote(remoteConfig)
	if err == scs.ErrNoDefault {
		sylog.Warningf("No default remote in use, falling back to: %v", keyServerURI)
		return
	} else if err != nil {
		sylog.Fatalf("Unable to load remote configuration: %v", err)
	}

	authToken = endpoint.Token
	uri, err := endpoint.GetServiceURI("keystore")
	if err != nil {
		sylog.Fatalf("Unable to get key service URI: %v", err)
	}
	keyServerURI = uri
}

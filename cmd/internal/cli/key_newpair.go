// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/sylabs/singularity/internal/pkg/util/interactive"

	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/sypgp"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

var (
	keyNewPairName     string
	KeyNewPairNameFlag = &cmdline.Flag{
		ID:           "KeyNewPairNameFlag",
		Value:        &keyNewPairName,
		DefaultValue: "",
		Name:         "name",
		ShortHand:    "N",
		Usage:        "key owner name",
	}

	keyNewPairEmail     string
	KeyNewPairEmailFlag = &cmdline.Flag{
		ID:           "KeyNewPairEmailFlag",
		Value:        &keyNewPairEmail,
		DefaultValue: "",
		Name:         "email",
		ShortHand:    "E",
		Usage:        "key owner email",
	}

	keyNewPairComment     string
	KeyNewPairCommentFlag = &cmdline.Flag{
		ID:           "KeyNewPairCommentFlag",
		Value:        &keyNewPairComment,
		DefaultValue: "",
		Name:         "comment",
		ShortHand:    "C",
		Usage:        "key comment",
	}

	keyNewPairPassword     string
	KeyNewPairPasswordFlag = &cmdline.Flag{
		ID:           "KeyNewPairPasswordFlag",
		Value:        &keyNewPairPassword,
		DefaultValue: "",
		Name:         "password",
		ShortHand:    "P",
		Usage:        "key password",
	}

	keyNewPairPush     bool
	KeyNewPairPushFlag = &cmdline.Flag{
		ID:           "KeyNewPairPushFlag",
		Value:        &keyNewPairPush,
		DefaultValue: false,
		Name:         "push",
		ShortHand:    "U",
		Usage:        "specify to push the public key to the remote keystore (default true)",
	}

	// KeyNewPairCmd is 'singularity key newpair' and generate a new OpenPGP key pair
	KeyNewPairCmd = &cobra.Command{
		Args:                  cobra.ExactArgs(0),
		DisableFlagsInUseLine: true,
		PreRun:                sylabsToken,
		Run:                   runNewPairCmd,
		Use:                   docs.KeyNewPairUse,
		Short:                 docs.KeyNewPairShort,
		Long:                  docs.KeyNewPairLong,
		Example:               docs.KeyNewPairExample,
	}
)

type keyNewPairOptions struct {
	sypgp.GenKeyPairOptions
	PushToKeyStore bool
}

func runNewPairCmd(cmd *cobra.Command, args []string) {
	keyring := sypgp.NewHandle("")

	opts, err := collectInput(cmd)
	if err != nil {
		sylog.Errorf("could not collect user input: %v", err)
		os.Exit(2)
	}
	opts.KeyLength = keyNewpairBitLength

	fmt.Printf("Generating Entity and OpenPGP Key Pair... ")
	key, err := keyring.GenKeyPair(opts.GenKeyPairOptions)
	if err != nil {
		sylog.Errorf("creating newpair failed: %v", err)
		os.Exit(2)
	}
	fmt.Printf("done\n")

	if !opts.PushToKeyStore {
		fmt.Printf("NOT pushing newly created key to: %s\n", keyServerURI)
		return
	}

	// Only connect to the endpoint if we are pushing the key.
	handleKeyNewPairEndpoint()
	if err := sypgp.PushPubkey(http.DefaultClient, key, keyServerURI, authToken); err != nil {
		fmt.Printf("Failed to push newly created key to keystore: %s\n", err)
	} else {
		fmt.Printf("Key successfully pushed to: %s\n", keyServerURI)
	}
}

// collectInput collects passed flags, for missed parameters will ask user input.
func collectInput(cmd *cobra.Command) (*keyNewPairOptions, error) {
	var genOpts keyNewPairOptions

	// check flags
	if cmd.Flags().Changed(KeyNewPairNameFlag.Name) {
		genOpts.Name = keyNewPairName
	} else {
		n, err := interactive.AskQuestion("Enter your name (e.g., John Doe) : ")
		if err != nil {
			return nil, err
		}

		genOpts.Name = n
	}

	if cmd.Flags().Changed(KeyNewPairEmailFlag.Name) {
		genOpts.Email = keyNewPairEmail
	} else {
		e, err := interactive.AskQuestion("Enter your email address (e.g., john.doe@example.com) : ")
		if err != nil {
			return nil, err
		}
		genOpts.Email = e
	}

	if cmd.Flags().Changed(KeyNewPairCommentFlag.Name) {
		genOpts.Comment = keyNewPairComment
	} else {
		c, err := interactive.AskQuestion("Enter optional comment (e.g., development keys) : ")
		if err != nil {
			return nil, err
		}
		genOpts.Comment = c
	}

	if cmd.Flags().Changed(KeyNewPairPasswordFlag.Name) {
		genOpts.Password = keyNewPairPassword
	} else {
		// get a password
		p, err := interactive.GetPassphrase("Enter a passphrase : ", 3)
		if err != nil {
			return nil, err
		}
		if p == "" {
			a, err := interactive.AskYNQuestion("n", "WARNING: if there is no password set, your key is not secure. Do you want to continue? [y/n] ")
			if err != nil {
				return nil, err
			}

			if a == "n" {
				return nil, errors.New("empty passphrase")
			}

		}

		genOpts.Password = p
	}

	if cmd.Flags().Changed(KeyNewPairPushFlag.Name) {
		genOpts.PushToKeyStore = keyNewPairPush
	} else {
		a, err := interactive.AskYNQuestion("y", "Would you like to push it to the keystore? [Y,n] ")
		if err != nil {
			return nil, err
		}

		if a == "y" {
			genOpts.PushToKeyStore = true
		}
	}

	return &genOpts, nil
}

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

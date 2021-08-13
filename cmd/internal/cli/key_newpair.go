// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2017-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/hpcng/singularity/docs"
	"github.com/hpcng/singularity/internal/pkg/remote/endpoint"
	"github.com/hpcng/singularity/internal/pkg/util/interactive"
	"github.com/hpcng/singularity/pkg/cmdline"
	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/hpcng/singularity/pkg/sypgp"
	"github.com/spf13/cobra"
)

var (
	keyNewPairName     string
	keyNewPairNameFlag = &cmdline.Flag{
		ID:           "KeyNewPairNameFlag",
		Value:        &keyNewPairName,
		DefaultValue: "",
		Name:         "name",
		ShortHand:    "N",
		Usage:        "key owner name",
	}

	keyNewPairEmail     string
	keyNewPairEmailFlag = &cmdline.Flag{
		ID:           "KeyNewPairEmailFlag",
		Value:        &keyNewPairEmail,
		DefaultValue: "",
		Name:         "email",
		ShortHand:    "E",
		Usage:        "key owner email",
	}

	keyNewPairComment     string
	keyNewPairCommentFlag = &cmdline.Flag{
		ID:           "KeyNewPairCommentFlag",
		Value:        &keyNewPairComment,
		DefaultValue: "",
		Name:         "comment",
		ShortHand:    "C",
		Usage:        "key comment",
	}

	keyNewPairPassword     string
	keyNewPairPasswordFlag = &cmdline.Flag{
		ID:           "KeyNewPairPasswordFlag",
		Value:        &keyNewPairPassword,
		DefaultValue: "",
		Name:         "password",
		ShortHand:    "P",
		Usage:        "key password",
	}

	keyNewPairPush     bool
	keyNewPairPushFlag = &cmdline.Flag{
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
	ctx := context.TODO()

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
		fmt.Println("NOT pushing newly created key to keystore")
		return
	}

	// Only connect to the endpoint if we are pushing the key.
	co, err := getKeyserverClientOpts(keyServerURI, endpoint.KeyserverPushOp)
	if err != nil {
		sylog.Fatalf("Keyserver client failed: %s", err)
	}

	if err := sypgp.PushPubkey(ctx, key, co...); err != nil {
		fmt.Printf("Failed to push newly created key to keystore: %s\n", err)
	} else {
		fmt.Println("Key successfully pushed to keystore")
	}
}

// collectInput collects passed flags, for missed parameters will ask user input.
func collectInput(cmd *cobra.Command) (*keyNewPairOptions, error) {
	var genOpts keyNewPairOptions

	// check flags
	if cmd.Flags().Changed(keyNewPairNameFlag.Name) {
		genOpts.Name = keyNewPairName
	} else {
		n, err := interactive.AskQuestion("Enter your name (e.g., John Doe) : ")
		if err != nil {
			return nil, err
		}

		genOpts.Name = n
	}

	if cmd.Flags().Changed(keyNewPairEmailFlag.Name) {
		genOpts.Email = keyNewPairEmail
	} else {
		e, err := interactive.AskQuestion("Enter your email address (e.g., john.doe@example.com) : ")
		if err != nil {
			return nil, err
		}
		genOpts.Email = e
	}

	if cmd.Flags().Changed(keyNewPairCommentFlag.Name) {
		genOpts.Comment = keyNewPairComment
	} else {
		c, err := interactive.AskQuestion("Enter optional comment (e.g., development keys) : ")
		if err != nil {
			return nil, err
		}
		genOpts.Comment = c
	}

	if cmd.Flags().Changed(keyNewPairPasswordFlag.Name) {
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

	if cmd.Flags().Changed(keyNewPairPushFlag.Name) {
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

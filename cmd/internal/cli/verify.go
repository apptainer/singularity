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
	"github.com/sylabs/singularity/pkg/signing"
)

var (
	sifGroupID  uint32 // -g groupid specification
	sifDescID   uint32 // -i id specification
	localVerify bool   // -l flag
)

// -u|--url
var verifyServerURIFlag = cmdline.Flag{
	ID:           "verifyServerURIFlag",
	Value:        &keyServerURI,
	DefaultValue: defaultKeyServer,
	Name:         "url",
	ShortHand:    "u",
	Usage:        "key server URL",
	EnvKeys:      []string{"URL"},
}

// -g|--groupid
var verifySifGroupIDFlag = cmdline.Flag{
	ID:           "verifySifGroupIDFlag",
	Value:        &sifGroupID,
	DefaultValue: uint32(0),
	Name:         "groupid",
	ShortHand:    "g",
	Usage:        "group ID to be verified",
}

// -i|--id
var verifySifDescIDFlag = cmdline.Flag{
	ID:           "verifySifDescIDFlag",
	Value:        &sifDescID,
	DefaultValue: uint32(0),
	Name:         "id",
	ShortHand:    "i",
	Usage:        "descriptor ID to be verified",
}

// -l|--local
var verifyLocalFlag = cmdline.Flag{
	ID:           "verifyLocalFlag",
	Value:        &localVerify,
	DefaultValue: false,
	Name:         "local",
	ShortHand:    "l",
	Usage:        "only verify with local keys",
	EnvKeys:      []string{"LOCAL_VERIFY"},
}

func init() {
	cmdManager.RegisterCmd(VerifyCmd)

	cmdManager.RegisterFlagForCmd(&verifyServerURIFlag, VerifyCmd)
	cmdManager.RegisterFlagForCmd(&verifySifGroupIDFlag, VerifyCmd)
	cmdManager.RegisterFlagForCmd(&verifySifDescIDFlag, VerifyCmd)
	cmdManager.RegisterFlagForCmd(&verifyLocalFlag, VerifyCmd)
}

// VerifyCmd singularity verify
var VerifyCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),
	PreRun:                sylabsToken,

	Run: func(cmd *cobra.Command, args []string) {
		// dont need to resolve remote endpoint
		if !localVerify {
			handleVerifyFlags(cmd)
		}

		// args[0] contains image path
		doVerifyCmd(args[0], keyServerURI)
	},

	Use:     docs.VerifyUse,
	Short:   docs.VerifyShort,
	Long:    docs.VerifyLong,
	Example: docs.VerifyExample,
}

func doVerifyCmd(cpath, url string) {
	if sifGroupID != 0 && sifDescID != 0 {
		sylog.Fatalf("only one of -i or -g may be set")
	}

	var isGroup bool
	var id uint32
	if sifGroupID != 0 {
		isGroup = true
		id = sifGroupID
	} else {
		id = sifDescID
	}

	localKeyOk, err := signing.Verify(cpath, url, id, isGroup, authToken, localVerify, false, false)
	if err != nil {
		sylog.Fatalf("Failed to verify: %s: %v", cpath, err)
	}
	if localKeyOk {
		os.Exit(1)
	}
}

func handleVerifyFlags(cmd *cobra.Command) {
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
	if !cmd.Flags().Lookup("url").Changed {
		uri, err := endpoint.GetServiceURI("keystore")
		if err != nil {
			sylog.Fatalf("Unable to get library service URI: %v", err)
		}
		keyServerURI = uri
	}
}

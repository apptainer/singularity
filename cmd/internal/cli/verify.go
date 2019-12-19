// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"fmt"
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
	jsonVerify  bool   // -j flag
	verifyAll   bool
)

// -u|--url
var verifyServerURIFlag = cmdline.Flag{
	ID:           "verifyServerURIFlag",
	Value:        &keyServerURI,
	DefaultValue: defaultKeyServer,
	Name:         "url",
	ShortHand:    "u",
	Usage:        "specify a URL for a key server",
	EnvKeys:      []string{"URL"},
}

// -g|--group-id
var verifySifGroupIDFlag = cmdline.Flag{
	ID:           "verifySifGroupIDFlag",
	Value:        &sifGroupID,
	DefaultValue: uint32(0),
	Name:         "group-id",
	ShortHand:    "g",
	Usage:        "verify all partitions in the specified group (default non)",
}

// --groupid (deprecated)
var verifyOldSifGroupIDFlag = cmdline.Flag{
	ID:           "verifyOldSifGroupIDFlag",
	Value:        &sifGroupID,
	DefaultValue: uint32(0),
	Name:         "groupid",
	Usage:        "group ID to be verified",
	Deprecated:   "use '--group-id'",
}

// -i|--sif-id
var verifySifDescSifIDFlag = cmdline.Flag{
	ID:           "verifySifDescSifIDFlag",
	Value:        &sifDescID,
	DefaultValue: uint32(0),
	Name:         "sif-id",
	ShortHand:    "i",
	Usage:        "verify a single partition with the specified ID (default system-partition)",
}

// --id (deprecated)
var verifySifDescIDFlag = cmdline.Flag{
	ID:           "verifySifDescIDFlag",
	Value:        &sifDescID,
	DefaultValue: uint32(0),
	Name:         "id",
	Usage:        "descriptor ID to be verified",
	Deprecated:   "use '--sif-id'",
}

// -l|--local
var verifyLocalFlag = cmdline.Flag{
	ID:           "verifyLocalFlag",
	Value:        &localVerify,
	DefaultValue: false,
	Name:         "local",
	ShortHand:    "l",
	Usage:        "only verify with local key(s) in keyring",
	EnvKeys:      []string{"LOCAL_VERIFY"},
}

// -j|--json
var verifyJSONFlag = cmdline.Flag{
	ID:           "verifyJsonFlag",
	Value:        &jsonVerify,
	DefaultValue: false,
	Name:         "json",
	ShortHand:    "j",
	Usage:        "output json",
}

// -a|--all
var verifyAllFlag = cmdline.Flag{
	ID:           "verifyAllFlag",
	Value:        &verifyAll,
	DefaultValue: false,
	Name:         "all",
	ShortHand:    "a",
	Usage:        "verify all non-signature partitions",
}

func init() {
	cmdManager.RegisterCmd(VerifyCmd)

	cmdManager.RegisterFlagForCmd(&verifyServerURIFlag, VerifyCmd)
	cmdManager.RegisterFlagForCmd(&verifySifGroupIDFlag, VerifyCmd)
	cmdManager.RegisterFlagForCmd(&verifyOldSifGroupIDFlag, VerifyCmd)
	cmdManager.RegisterFlagForCmd(&verifySifDescSifIDFlag, VerifyCmd)
	cmdManager.RegisterFlagForCmd(&verifySifDescIDFlag, VerifyCmd)
	cmdManager.RegisterFlagForCmd(&verifyLocalFlag, VerifyCmd)
	cmdManager.RegisterFlagForCmd(&verifyJSONFlag, VerifyCmd)
	cmdManager.RegisterFlagForCmd(&verifyAllFlag, VerifyCmd)
}

// VerifyCmd singularity verify
var VerifyCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),
	PreRun:                sylabsToken,

	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.TODO()

		if f, err := os.Stat(args[0]); os.IsNotExist(err) {
			sylog.Fatalf("No such file or directory: %s", args[0])
		} else if f.IsDir() {
			sylog.Fatalf("File is a directory: %s", args[0])
		}

		// args[0] contains image path
		doVerifyCmd(ctx, cmd, args[0], keyServerURI)
	},

	Use:     docs.VerifyUse,
	Short:   docs.VerifyShort,
	Long:    docs.VerifyLong,
	Example: docs.VerifyExample,
}

func doVerifyCmd(ctx context.Context, cmd *cobra.Command, cpath, url string) {
	id, isGroup, err := checkImageAndFlags(cmd, cpath, sifDescID, sifGroupID, verifyAll)
	if err != nil {
		sylog.Fatalf("%s", err)
	}

	// Dont need to resolve remote endpoint.
	if !localVerify {
		handleVerifyFlags(cmd)
	}

	author, _, err := signing.Verify(ctx, cpath, url, id, isGroup, verifyAll, authToken, localVerify, jsonVerify)
	fmt.Printf("%s", author)
	if err == signing.ErrVerificationFail {
		sylog.Fatalf("Failed to verify: %s", cpath)
	} else if err != nil {
		sylog.Fatalf("Failed to verify: %s: %s", cpath, err)
	}
	sylog.Infof("Container verified: %s", cpath)
}

// checkImageAndFlags verifies that the SIF image and SIF descriptor / group
// ID flags provided are usable, and returns which SIF ID to sign or verify:
//  - checks that a descrID and groupID are not being used together
//  - checks that a descrID or groupID are not being used with a request to verify all
//  - returns an error if flags are not usable
//  - returns an ID to sign or verify, and true if it is a group ID / false if it is a descriptor ID
func checkImageAndFlags(cmd *cobra.Command, cpath string, descrID, groupID uint32, all bool) (uint32, bool, error) {
	// First ensure the image is there.
	if finfo, err := os.Stat(cpath); os.IsNotExist(err) || finfo.IsDir() {
		return 0, false, fmt.Errorf("Failed to open: %s: %s", cpath, err)
	}

	// Group id should start at 1.
	if cmd.Flag(verifySifGroupIDFlag.Name).Changed && groupID == 0 {
		return 0, false, fmt.Errorf("invalid group id")
	}

	// Descriptor id should start at 1.
	if cmd.Flag(verifySifDescSifIDFlag.Name).Changed && descrID == 0 {
		return 0, false, fmt.Errorf("invalid descriptor id")
	}
	if cmd.Flag(verifySifDescIDFlag.Name).Changed && descrID == 0 {
		sylog.Fatalf("invalid descriptor id")
	}

	if groupID != 0 && descrID != 0 {
		return 0, false, fmt.Errorf("only one of -i or -g may be set")
	}

	var isGroup bool
	var id uint32
	if groupID != 0 {
		isGroup = true
		id = groupID
	} else {
		id = descrID
	}

	if (id != 0 || isGroup) && all {
		return 0, false, fmt.Errorf("'--all' not compatible with '--sif-id' or '--group-id'")
	}

	return id, isGroup, nil
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

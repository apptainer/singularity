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
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/signing"
)

var (
	sifGroupID uint32 // -g groupid specification
	sifDescID  uint32 // -i id specification
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

func init() {
	cmdManager.RegisterCmd(VerifyCmd, false)

	flagManager.RegisterCmdFlag(&verifyServerURIFlag, VerifyCmd)
	flagManager.RegisterCmdFlag(&verifySifGroupIDFlag, VerifyCmd)
	flagManager.RegisterCmdFlag(&verifySifDescIDFlag, VerifyCmd)
}

// VerifyCmd singularity verify
var VerifyCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),
	PreRun:                sylabsToken,

	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains image path
		fmt.Printf("Verifying image: %s\n", args[0])
		if err := doVerifyCmd(args[0], keyServerURI); err != nil {
			sylog.Errorf("verification failed: %s", err)
			os.Exit(2)
		}
	},

	Use:     docs.VerifyUse,
	Short:   docs.VerifyShort,
	Long:    docs.VerifyLong,
	Example: docs.VerifyExample,
}

func doVerifyCmd(cpath, url string) error {
	if sifGroupID != 0 && sifDescID != 0 {
		return fmt.Errorf("only one of -i or -g may be set")
	}

	var isGroup bool
	var id uint32
	if sifGroupID != 0 {
		isGroup = true
		id = sifGroupID
	} else {
		id = sifDescID
	}

	return signing.Verify(cpath, url, id, isGroup, authToken, false)
}

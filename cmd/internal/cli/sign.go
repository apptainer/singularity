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
	privKey int // -k encryption key (index from 'keys list') specification
)

// -u|--url
var signServerURIFlag = cmdline.Flag{
	ID:           "signServerURIFlag",
	Value:        &keyServerURI,
	DefaultValue: defaultKeyServer,
	Name:         "url",
	ShortHand:    "u",
	Usage:        "key server URL",
	EnvKeys:      []string{"URL"},
}

// -g|--groupid
var signSifGroupIDFlag = cmdline.Flag{
	ID:           "signSifGroupIDFlag",
	Value:        &sifGroupID,
	DefaultValue: uint32(0),
	Name:         "groupid",
	ShortHand:    "g",
	Usage:        "group ID to be signed",
}

// -i|--id
var signSifDescIDFlag = cmdline.Flag{
	ID:           "signSifDescIDFlag",
	Value:        &sifDescID,
	DefaultValue: uint32(0),
	Name:         "id",
	ShortHand:    "i",
	Usage:        "descriptor ID to be signed",
}

// -k|--keyidx
var signKeyIdxFlag = cmdline.Flag{
	ID:           "signKeyIdxFlag",
	Value:        &privKey,
	DefaultValue: -1,
	Name:         "keyidx",
	ShortHand:    "k",
	Usage:        "private key to use (index from 'keys list')",
}

func init() {
	cmdManager.RegisterCmd(SignCmd, false)

	flagManager.RegisterCmdFlag(&signServerURIFlag, SignCmd)
	flagManager.RegisterCmdFlag(&signSifGroupIDFlag, SignCmd)
	flagManager.RegisterCmdFlag(&signSifDescIDFlag, SignCmd)
	flagManager.RegisterCmdFlag(&signKeyIdxFlag, SignCmd)
}

// SignCmd singularity sign
var SignCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),
	PreRun:                sylabsToken,

	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains image path
		fmt.Printf("Signing image: %s\n", args[0])
		if err := doSignCmd(args[0], keyServerURI); err != nil {
			sylog.Errorf("signing container failed: %s", err)
			os.Exit(2)
		}
		fmt.Printf("Signature created and applied to %v\n", args[0])
	},

	Use:     docs.SignUse,
	Short:   docs.SignShort,
	Long:    docs.SignLong,
	Example: docs.SignExample,
}

func doSignCmd(cpath, url string) error {
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

	return signing.Sign(cpath, url, id, isGroup, privKey, authToken)
}

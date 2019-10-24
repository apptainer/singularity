// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/signing"
)

var (
	privKey int // -k encryption key (index from 'keys list') specification
	signAll bool
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

// -i| --sif-id
var signSifDescSifIDFlag = cmdline.Flag{
	ID:           "signSifDescSifIDFlag",
	Value:        &sifDescID,
	DefaultValue: uint32(0),
	Name:         "sif-id",
	ShortHand:    "i",
	Usage:        "descriptor ID to be signed (default system-partition)",
}

// --id (deprecated)
var signSifDescIDFlag = cmdline.Flag{
	ID:           "signSifDescIDFlag",
	Value:        &sifDescID,
	DefaultValue: uint32(0),
	Name:         "id",
	Usage:        "descriptor ID to be signed",
	Deprecated:   "use '--sif-id'",
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

// -a|--all
var signAllFlag = cmdline.Flag{
	ID:           "signAllFlag",
	Value:        &signAll,
	DefaultValue: false,
	Name:         "all",
	ShortHand:    "a",
	Usage:        "sign all non-signature partitions in a SIF",
}

func init() {
	cmdManager.RegisterCmd(SignCmd)

	cmdManager.RegisterFlagForCmd(&signServerURIFlag, SignCmd)
	cmdManager.RegisterFlagForCmd(&signSifGroupIDFlag, SignCmd)
	cmdManager.RegisterFlagForCmd(&signSifDescSifIDFlag, SignCmd)
	cmdManager.RegisterFlagForCmd(&signSifDescIDFlag, SignCmd)
	cmdManager.RegisterFlagForCmd(&signKeyIdxFlag, SignCmd)
	cmdManager.RegisterFlagForCmd(&signAllFlag, SignCmd)
}

// SignCmd singularity sign
var SignCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains image path
		fmt.Printf("Signing image: %s\n", args[0])
		doSignCmd(cmd, args[0])
	},

	Use:     docs.SignUse,
	Short:   docs.SignShort,
	Long:    docs.SignLong,
	Example: docs.SignExample,
}

func doSignCmd(cmd *cobra.Command, cpath string) {
	// Group id should start at 1.
	if cmd.Flag(verifySifGroupIDFlag.Name).Changed && sifGroupID == 0 {
		sylog.Fatalf("invalid group id")
	}

	// Descriptor id should start at 1.
	if cmd.Flag(verifySifDescSifIDFlag.Name).Changed && sifDescID == 0 {
		sylog.Fatalf("invalid descriptor id")
	}
	if cmd.Flag(verifySifDescIDFlag.Name).Changed && sifDescID == 0 {
		sylog.Fatalf("invalid descriptor id")
	}

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

	if (id != 0 || isGroup) && signAll {
		sylog.Fatalf("'--all' not compatible with '--sif-id' or '--groupid'")
	}

	if err := signing.Sign(cpath, id, isGroup, signAll, privKey); err != nil {
		sylog.Fatalf("Failed to sign container: %s", err)
	}
	fmt.Printf("Signature created and applied to %s\n", cpath)
}

// Copyright (c) 2017-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
)

var (
	privKey int // -k encryption key (index from 'keys list') specification
	signAll bool
)

// -g|--group-id
var signSifGroupIDFlag = cmdline.Flag{
	ID:           "signSifGroupIDFlag",
	Value:        &sifGroupID,
	DefaultValue: uint32(0),
	Name:         "group-id",
	ShortHand:    "g",
	Usage:        "sign objects with the specified group ID",
}

// --groupid (deprecated)
var signOldSifGroupIDFlag = cmdline.Flag{
	ID:           "signOldSifGroupIDFlag",
	Value:        &sifGroupID,
	DefaultValue: uint32(0),
	Name:         "groupid",
	Usage:        "sign objects with the specified group ID",
	Deprecated:   "use '--group-id'",
}

// -i| --sif-id
var signSifDescSifIDFlag = cmdline.Flag{
	ID:           "signSifDescSifIDFlag",
	Value:        &sifDescID,
	DefaultValue: uint32(0),
	Name:         "sif-id",
	ShortHand:    "i",
	Usage:        "sign object with the specified ID",
}

// --id (deprecated)
var signSifDescIDFlag = cmdline.Flag{
	ID:           "signSifDescIDFlag",
	Value:        &sifDescID,
	DefaultValue: uint32(0),
	Name:         "id",
	Usage:        "sign object with the specified ID",
	Deprecated:   "use '--sif-id'",
}

// -k|--keyidx
var signKeyIdxFlag = cmdline.Flag{
	ID:           "signKeyIdxFlag",
	Value:        &privKey,
	DefaultValue: 0,
	Name:         "keyidx",
	ShortHand:    "k",
	Usage:        "private key to use (index from 'key list')",
}

// -a|--all (deprecated)
var signAllFlag = cmdline.Flag{
	ID:           "signAllFlag",
	Value:        &signAll,
	DefaultValue: false,
	Name:         "all",
	ShortHand:    "a",
	Usage:        "sign all objects",
	Deprecated:   "now the default behavior",
}

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterCmd(SignCmd)

		cmdManager.RegisterFlagForCmd(&signSifGroupIDFlag, SignCmd)
		cmdManager.RegisterFlagForCmd(&signOldSifGroupIDFlag, SignCmd)
		cmdManager.RegisterFlagForCmd(&signSifDescSifIDFlag, SignCmd)
		cmdManager.RegisterFlagForCmd(&signSifDescIDFlag, SignCmd)
		cmdManager.RegisterFlagForCmd(&signKeyIdxFlag, SignCmd)
		cmdManager.RegisterFlagForCmd(&signAllFlag, SignCmd)
	})
}

// SignCmd singularity sign
var SignCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains image path
		doSignCmd(cmd, args[0])
	},

	Use:     docs.SignUse,
	Short:   docs.SignShort,
	Long:    docs.SignLong,
	Example: docs.SignExample,
}

func doSignCmd(cmd *cobra.Command, cpath string) {
	var opts []singularity.SignOpt

	// Set entity selector option, and ensure the entity is decrypted.
	var f sypgp.EntitySelector
	if cmd.Flag(signKeyIdxFlag.Name).Changed {
		f = selectEntityAtIndex(privKey)
	} else {
		f = selectEntityInteractive()
	}
	f = decryptSelectedEntityInteractive(f)
	opts = append(opts, singularity.OptSignEntitySelector(f))

	// Set group option, if applicable.
	if cmd.Flag(signSifGroupIDFlag.Name).Changed || cmd.Flag(signOldSifGroupIDFlag.Name).Changed {
		opts = append(opts, singularity.OptSignGroup(sifGroupID))
	}

	// Set object option, if applicable.
	if cmd.Flag(signSifDescSifIDFlag.Name).Changed || cmd.Flag(signSifDescIDFlag.Name).Changed {
		opts = append(opts, singularity.OptSignObjects(sifDescID))
	}

	// Sign the image.
	fmt.Printf("Signing image: %s\n", cpath)
	if err := singularity.Sign(cpath, opts...); err != nil {
		sylog.Fatalf("Failed to sign container: %s", err)
	}
	fmt.Printf("Signature created and applied to %s\n", cpath)
}

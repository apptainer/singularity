// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/sypgp"
)

var (
	secretKeyRemove   bool
	secretForceRemove bool
)

var keyRemoveSecretFlag = cmdline.Flag{
	ID:           "keyRemoveSecretFlag",
	Value:        &secretKeyRemove,
	DefaultValue: false,
	Name:         "secret",
	ShortHand:    "s",
	Usage:        "remove a secret key",
}

var keyRemoveForceFlag = cmdline.Flag{
	ID:           "keyRemoveForceFlag",
	Value:        &secretForceRemove,
	DefaultValue: false,
	Name:         "force",
	ShortHand:    "f",
	Usage:        "remove a secret key without prompting",
}

func init() {
	//	cmdManager.RegisterCmd(KeyRemoveCmd)

	cmdManager.RegisterFlagForCmd(&keyRemoveSecretFlag, KeyRemoveCmd)
	cmdManager.RegisterFlagForCmd(&keyRemoveForceFlag, KeyRemoveCmd)
}

// KeyRemoveCmd is `singularity key remove <fingerprint>' command
var KeyRemoveCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {

		err := sypgp.RemoveKey(secretKeyRemove, secretForceRemove, args[0])
		if err != nil {
			sylog.Fatalf("Unable to remove public key: %s", err)
		}

	},

	Use:     docs.KeyRemoveUse,
	Short:   docs.KeyRemoveShort,
	Long:    docs.KeyRemoveLong,
	Example: docs.KeyRemoveExample,
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/signing"
	"github.com/sylabs/singularity/docs"
)

var (
	privKey int // -k encryption key (index from 'keys list') specification
)

func init() {
	SignCmd.Flags().SetInterspersed(false)

	SignCmd.Flags().StringVarP(&keyServerURL, "url", "u", defaultKeysServer, "key server URL")
	SignCmd.Flags().SetAnnotation("url", "envkey", []string{"URL"})
	SignCmd.Flags().Uint32VarP(&sifGroupID, "groupid", "g", 0, "group ID to be signed")
	SignCmd.Flags().Uint32VarP(&sifDescID, "id", "i", 0, "descriptor ID to be signed")
	SignCmd.Flags().IntVarP(&privKey, "keyidx", "k", -1, "private key to use (index from 'keys list')")

	SingularityCmd.AddCommand(SignCmd)
}

// SignCmd singularity sign
var SignCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),
	PreRun:                sylabsToken,

	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains image path
		fmt.Printf("Signing image: %s\n", args[0])
		if err := doSignCmd(args[0], keyServerURL); err != nil {
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

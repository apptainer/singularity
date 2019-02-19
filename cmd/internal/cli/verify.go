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
	"github.com/sylabs/singularity/pkg/signing"
)

var (
	sifGroupID uint32 // -g groupid specification
	sifDescID  uint32 // -i id specification
)

func init() {
	VerifyCmd.Flags().SetInterspersed(false)

	VerifyCmd.Flags().StringVarP(&keyServerURL, "url", "u", defaultKeyServer, "key server URL")
	VerifyCmd.Flags().SetAnnotation("url", "envkey", []string{"URL"})
	VerifyCmd.Flags().Uint32VarP(&sifGroupID, "groupid", "g", 0, "group ID to be verified")
	VerifyCmd.Flags().Uint32VarP(&sifDescID, "id", "i", 0, "descriptor ID to be verified")
	SingularityCmd.AddCommand(VerifyCmd)
}

// VerifyCmd singularity verify
var VerifyCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),
	PreRun:                sylabsToken,

	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains image path
		fmt.Printf("Verifying image: %s\n", args[0])
		if err := doVerifyCmd(args[0], keyServerURL); err != nil {
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

	fmt.Println("cpath: ", cpath)
	fmt.Println("url: ", url)
	fmt.Println("id: ", id)
	fmt.Println("isGroup: ", isGroup)
	fmt.Println("autthtoken", authToken)


	return signing.Verify(cpath, url, id, isGroup, authToken, false)
}

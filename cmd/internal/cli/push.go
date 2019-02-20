// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	client "github.com/sylabs/singularity/pkg/client/library"
	"github.com/sylabs/singularity/pkg/signing"
)

var (
	// PushLibraryURI holds the base URI to a Sylabs library API instance
	PushLibraryURI string

	unauthenticatedPush bool
)

func init() {
	PushCmd.Flags().SetInterspersed(false)

	PushCmd.Flags().StringVar(&PushLibraryURI, "library", "https://library.sylabs.io", "the library to push to")
	PushCmd.Flags().SetAnnotation("library", "envkey", []string{"LIBRARY"})

	PushCmd.Flags().BoolVarP(&unauthenticatedPush, "allow-unauthenticated", "U", false, "dont check if the container is signed")
	PushCmd.Flags().SetAnnotation("allow-unauthenticated", "envkey", []string{"ALLOW-UNAUTHENTICATED"})

	SingularityCmd.AddCommand(PushCmd)
}

// PushCmd singularity push
var PushCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(2),
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		// Push to library requires a valid authToken
		if authToken != "" {
			if !unauthenticatedPush {
				imageSigned, err := signing.IsSigned(args[0], "", 0, false, authToken, true)
				if err != nil {
					sylog.Fatalf("Unable to verify container: %v", err)
				}
				if !imageSigned {
					fmt.Println("Visit here for instructions on how to sign a container : https://www.sylabs.io/guides/3.0/user-guide/signNverify.html#verifying-containers-from-the-container-library")
					sylog.Warningf("Your container is **NOT** signed! You REALLY should sign your container before pushing!")
					fmt.Printf("Do you really want to continue? [N/y] ")
					reader := bufio.NewReader(os.Stdin)
					input, err := reader.ReadString('\n')
					if err != nil {
						sylog.Fatalf("Error parsing input: %s", err)
					}
					if val := strings.Compare(strings.ToLower(input), "y\n"); val != 0 {
						fmt.Printf("Stoping upload.\n")
						os.Exit(3)
					}
				}
			} else {
				sylog.Warningf("Skipping container verifying")
			}

			err := client.UploadImage(args[0], args[1], PushLibraryURI, authToken, "No Description")
			if err != nil {
				sylog.Fatalf("%v\n", err)
			}
		} else {
			sylog.Fatalf("Couldn't push image to library: %v", authWarning)
		}
	},

	Use:     docs.PushUse,
	Short:   docs.PushShort,
	Long:    docs.PushLong,
	Example: docs.PushExample,
}

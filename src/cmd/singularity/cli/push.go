// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os/user"
	"path"

	"github.com/singularityware/singularity/src/pkg/libexec"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/spf13/cobra"
)

var (
	// PushLibraryURI holds the base URI to a Sylabs library API instance
	PushLibraryURI string
	// PushTokenFile holds the path to the sylabs auth token
	PushTokenFile string
)

func init() {
	usr, err := user.Current()
	if err != nil {
		sylog.Fatalf("Couldn't determine user home directory: %v", err)
	}

	defaultTokenFile := path.Join(usr.HomeDir, ".singularity", "sylabs-token")
	pushCmd.Flags().StringVar(&PushLibraryURI, "libraryuri", "https://library.sylabs.io", "")
	pushCmd.Flags().StringVar(&PushTokenFile, "tokenfile", defaultTokenFile, "path to the file holding your sylabs authentication token")
	singularityCmd.AddCommand(pushCmd)
}

var pushCmd = &cobra.Command{
	Use:  "push myimage.sif library://user/collection/container[:tag[,tag]...]",
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		libexec.PushImage(args[0], args[1], PushLibraryURI, PushTokenFile)
	},
}

/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"github.com/singularityware/singularity/src/pkg/libexec"
	"github.com/spf13/cobra"
)

var (
	PushLibraryURI string
)

func init() {
	pushCmd.Flags().StringVar(&PushLibraryURI, "libraryuri", "http://localhost:5150", "")
	singularityCmd.AddCommand(pushCmd)
}

var pushCmd = &cobra.Command{
	Use:  "push myimage.sif library://user/collection/container:tag",
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		libexec.PushImage(args[0], args[1], PushLibraryURI)
	},
}

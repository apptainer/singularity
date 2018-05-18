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
	// PullLibraryURI holds the base URI to a Sylabs library API instance
	PullLibraryURI string
)

func init() {
	pullCmd.Flags().StringVar(&PullLibraryURI, "libraryuri", "https://library.sylabs.io", "")
	pullCmd.Flags().BoolVarP(&Force, "force", "F", false, "overwrite an image file if it exists")
	singularityCmd.AddCommand(pullCmd)

}

var pullCmd = &cobra.Command{
	Use:  "pull [options] [myimage.sif] library://user/collection/container[:tag]",
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			libexec.PullImage(args[0], args[1], PullLibraryURI, Force)
			return
		}
		libexec.PullImage("", args[0], PullLibraryURI, Force)
	},
}

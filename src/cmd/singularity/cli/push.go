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

	"github.com/singularityware/singularity/docs"
)

var (
	// PushLibraryURI holds the base URI to a Sylabs library API instance
	PushLibraryURI string
)

func init() {
	pushCmd.Flags().SetInterspersed(false)
	SingularityCmd.AddCommand(pushCmd)

	pushCmd.Flags().StringVar(&PushLibraryURI, "libraryuri", "http://localhost:5150", "")
}

var pushCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args: cobra.ExactArgs(2),

	Run: func(cmd *cobra.Command, args []string) {
		libexec.PushImage(args[0], args[1], PushLibraryURI)
	},

	Use:     docs.PushUse,
	Short:   docs.PushShort,
	Long:    docs.PushLong,
	Example: docs.PushExample,
}

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

var pushUse string = `push [push options...] <container image> [library://[user[collection/[container[:tag]]]]]`

var pushShort string = `Push a container to a Library URI`

var pushLong string = `
The Singularity push command allows you to upload your sif image to a library
of your choosing`

var pushExample string = `
$ singularity push /home/user/my.sif library://user/collection/my.sif:latest
`

var (
	// PushLibraryURI holds the base URI to a Sylabs library API instance
	PushLibraryURI string
)

func init() {
	manHelp := func(c *cobra.Command, args []string) {
		docs.DispManPg("singularity-push")
	}

	pushCmd.Flags().SetInterspersed(false)
	pushCmd.SetHelpFunc(manHelp)
	SingularityCmd.AddCommand(pushCmd)

	pushCmd.Flags().StringVar(&PushLibraryURI, "libraryuri", "http://localhost:5150", "")
}

var pushCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args: cobra.ExactArgs(2),

	Run: func(cmd *cobra.Command, args []string) {
		libexec.PushImage(args[0], args[1], PushLibraryURI)
	},

	Use:     pushUse,
	Short:   pushShort,
	Long:    pushLong,
	Example: pushExample,
}

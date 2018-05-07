/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"github.com/singularityware/singularity/pkg/libexec"
	"github.com/spf13/cobra"

	"github.com/singularityware/singularity/docs"
)

var pullUse string = `pull [pull options...] [library://[user[collection/[<container>:tag]]]]`

var pullShort string = `
pull a contaier from a URI to PWD`

var pullLong string = `
SUPPORTED URIs:

    library: Pull an image from the currently configured library
    shub: Pull an image using python from Singularity Hub to /home/vagrant/versioned/singularity
    docker: Pull a docker image using python to /home/vagrant/versioned/singularity
`

var pullExample string = `
$ singularity pull docker://ubuntu:latest

$ singularity pull shub://vsoch/singularity-images
Found image vsoch/singularity-images:mongo
Downloading image... vsoch-singularity-images-mongo.img

$ singularity pull --name "meatballs.img" shub://vsoch/singularity-images
$ singularity pull --commit shub://vsoch/singularity-images
$ singularity pull --hash shub://vsoch/singularity-images
`

var (
	PullLibraryURI string
)

func init() {
	manHelp := func(c *cobra.Command, args []string) {
		docs.DispManPg("singularity-pull")
	}

	pullCmd.Flags().SetInterspersed(false)
	pullCmd.SetHelpFunc(manHelp)
	SingularityCmd.AddCommand(pullCmd)

	pullCmd.Flags().StringVar(&PullLibraryURI, "libraryuri", "http://localhost:5150", "")
}

var pullCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args: cobra.ExactArgs(2),

	Run: func(cmd *cobra.Command, args []string) {
		libexec.PullImage(args[0], args[1], PullLibraryURI)
	},

	Use:     pullUse,
	Short:   pullShort,
	Long:    pullLong,
	Example: pullExample,
}

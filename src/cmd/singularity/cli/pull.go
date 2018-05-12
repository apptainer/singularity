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

	// "github.com/singularityware/singularity/docs"
)

var pullUse string = `pull [pull options...] [library://[user[collection/[<container>:tag]]]]`

var pullShort string = `Pull a contianer from a URI`

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
	// PullLibraryURI holds the base URI to a Sylabs library API instance
	PullLibraryURI string
)

func init() {
	pullCmd.Flags().SetInterspersed(false)
	SingularityCmd.AddCommand(pullCmd)

	pullCmd.Flags().BoolVarP(&Force, "force", "F", false, "overwrite an image file if it exists")
	pullCmd.Flags().StringVar(&PullLibraryURI, "libraryuri", "http://localhost:5150", "")
}

var pullCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args: cobra.RangeArgs(1, 2),

	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			libexec.PullImage(args[0], args[1], PullLibraryURI, Force)
			return
		}
		libexec.PullImage("", args[0], PullLibraryURI, Force)
	},

	Use:     pullUse,
	Short:   pullShort,
	Long:    pullLong,
	Example: pullExample,
}

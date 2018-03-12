/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build cmd local vars
var (
	Sandbox  bool
	Writable bool
	Force    bool
	NoTest   bool
	Section  string
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use: "build",
	Run: func(cmd *cobra.Command, args []string) {
		if silent {
			fmt.Println("Silent!")
		}

		if Sandbox {
			fmt.Println("Sandbox!")
		}
	},
	TraverseChildren: true,
}

func init() {
	buildCmd.Flags().SetInterspersed(false)
	singularityCmd.AddCommand(buildCmd)

	buildCmd.Flags().BoolVarP(&Sandbox, "sandbox", "s", false, "")
	buildCmd.Flags().StringVar(&Section, "section", "", "")
	buildCmd.Flags().BoolVarP(&Writable, "writable", "w", false, "")
	buildCmd.Flags().BoolVarP(&Force, "force", "f", false, "")
	buildCmd.Flags().BoolVarP(&NoTest, "notest", "T", false, "")
}

/*
	buildCmd.SetHelpTemplate(`
	The build command compiles a container per a recipe (definition file) or based
	on a URI, location, or archive.

	CONTAINER PATH:
		When Singularity builds the container, the output format can be one of
		multiple formats:

			default:    The compressed Singularity read only image format (default)
			libray:		The container image will be stored on the libray after the
						build process
			sandbox:    This is a read-write container within a directory structure
			writable:   Legacy writable image format

		note: A common workflow is to use the "sandbox" mode for development of the
		container, and then build it as a default Singularity image  when done.
		This format can not be modified.

	BUILD SPEC TARGET:
		The build spec target is a Singularity recipe, local image, archive, or URI
		that can be used to create a Singularity container. Several different
		local target formats exist:

			def file  : This is a recipe for building a container (examples below)
			directory:  A directory structure containing a (ch)root file system
			image:      A local image on your machine (will convert to squashfs if
						it is legacy or writable format)
			tar/tar.gz: An archive file which contains the above directory format
						(must have .tar in the filename!)

		Targets can also be remote and defined by a URI of the following formats:

			shub://     Build from a Singularity registry (Singularity Hub default)
			docker://   This points to a Docker registry (Docker Hub default)

	BUILD OPTIONS:
		-s|--sandbox    Build a sandbox rather then a read only compressed image
		-w|--writable   Build a writable image
		-f|--force   Force a rebootstrap of a base OS (note: this does not
						delete what is currently in the image, just causes the core
						to be reinstalled)
		-T|--notest     Bootstrap without running tests in %test section
		-s|--section    Only run a given section within the recipe file (setup,
						post, files, environment, test, labels, none)

	CHECKS OPTIONS:
		-c|--checks    enable checks
		-t|--tag       specify a check tag (not default)
		-l|--low       Specify low threshold (all checks, default)
		-m|--med       Perform medium and high checks
		-h|--high      Perform only checks at level high

	See singularity check --help for available tags
`)*/

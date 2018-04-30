/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"fmt"
	"os"

	"github.com/singularityware/singularity/src/pkg/build"
	"github.com/spf13/cobra"
)

var (
	Remote    bool
	RemoteURL string
	Sandbox   bool
	Writable  bool
	Force     bool
	NoTest    bool
	Sections  []string
)

func init() {
	buildCmd.Flags().SetInterspersed(false)
	singularityCmd.AddCommand(buildCmd)

	buildCmd.Flags().BoolVarP(&Sandbox, "sandbox", "s", false, "Build image as sandbox format (chroot directory structure)")
	buildCmd.Flags().StringSliceVar(&Sections, "section", []string{}, "Only run specific section(s) of deffile")
	buildCmd.Flags().BoolVarP(&Writable, "writable", "w", false, "Build image as writable (SIF with writable internal overlay)")
	buildCmd.Flags().BoolVarP(&Force, "force", "f", false, "")
	buildCmd.Flags().BoolVarP(&NoTest, "notest", "T", false, "")
	buildCmd.Flags().BoolVarP(&Remote, "remote", "r", false, "Build image remotely")
	buildCmd.Flags().StringVar(&RemoteURL, "remote-url", "localhost:5050", "Specify the URL of the remote builder")
}

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:  "build <image path> <build spec>",
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var def build.Definition
		var b build.Builder
		var err error

		if silent {
			fmt.Println("Silent!")
		}

		if Sandbox {
			fmt.Println("Sandbox!")
		}

		if ok, err := build.IsValidURI(args[1]); ok && err == nil {
			// URI passed as arg[1]
			def, err = build.NewDefinitionFromURI(args[1])
			if err != nil {
				fmt.Println("Unable to parse URI %s: ", args[1], err)
				os.Exit(1)
			}
		} else if !ok && err == nil {
			// Non-URI passed as arg[1]
			defFile, err := os.Open(args[1])
			if err != nil {
				fmt.Println("Unable to open file %s: ", args[1], err)
				os.Exit(1)
			}

			def, err = build.ParseDefinitionFile(defFile)
			if err != nil {
				fmt.Println("Failed to parse definition file %s: ", args[1], err)
				os.Exit(1)
			}
		} else {
			// Error
			fmt.Println("Unable to parse %s: ", args[1], err)
			os.Exit(1)
		}

		if Remote {
			b = build.NewRemoteBuilder(args[0], def, false, RemoteURL)

		} else {
			b, err = build.NewSifBuilder(args[0], def)
			if err != nil {
				fmt.Println("Failed to create SifBuilder object: ", err)
				os.Exit(1)
			}
		}

		if err := b.Build(); err != nil {
			fmt.Println("Failed to build image: ", err)
			os.Exit(1)
		}

		/*
			if Remote {
				doRemoteBuild(args[0], args[1])
			} else {
				if ok, err := build.IsValidURI(args[1]); ok && err == nil {
					u := strings.SplitN(args[1], "://", 2)
					b, err := build.NewSifBuilderFromURI(args[0], args[1])
					if err != nil {
						glog.Errorf("Image build system encountered an error: %s\n", err)
						return
					}
					b.Build()
				} else {
					glog.Fatalf("%s", err)
				}
			}*/

	},
	TraverseChildren: true,
}

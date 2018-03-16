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

	"github.com/golang/glog"
	"github.com/singularityware/singularity/pkg/build"
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

func doRemoteBuild(imagePath string, defPath string) {
	// Parse the Deffile into json
	defFile, err := os.Open(defPath)
	if err != nil {
		glog.Fatal(err)
	}

	definition, err := build.ParseDefinitionFile(defFile)
	if err != nil {
		glog.Fatal(err)
	}

	b := build.NewRemoteBuilder(imagePath, definition, false, RemoteURL)
	b.Build()
}

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
		if silent {
			fmt.Println("Silent!")
		}

		if Sandbox {
			fmt.Println("Sandbox!")
		}

		if Remote {
			doRemoteBuild(args[0], args[1])
		} else {
			if ok, err := build.IsValidURI(args[1]); ok && err == nil {
				b, err := build.NewCachedBuilder(args[0], args[1])
				if err != nil {
					glog.Errorf("Image build system encountered an error: %s\n", err)
					return
				}
				b.Build()
			} else {
				glog.Fatalf("%s", err)
			}
		}

	},
	TraverseChildren: true,
}

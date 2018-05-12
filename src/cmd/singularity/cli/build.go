/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/singularityware/singularity/src/pkg/build"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/spf13/cobra"

	"github.com/singularityware/singularity/docs"
)

var (
	Remote    bool
	RemoteURL string
	AuthToken string
	Sandbox   bool
	Writable  bool
	Force     bool
	NoTest    bool
	Sections  []string
)

func init() {
	BuildCmd.Flags().SetInterspersed(false)
	SingularityCmd.AddCommand(BuildCmd)

	BuildCmd.Flags().BoolVarP(&Sandbox, "sandbox", "s", false, "Build image as sandbox format (chroot directory structure)")
	BuildCmd.Flags().StringSliceVar(&Sections, "section", []string{}, "Only run specific section(s) of deffile (setup, post, files, environment, test, labels, none)")
	BuildCmd.Flags().BoolVarP(&Writable, "writable", "w", false, "Build image as writable (SIF with writable internal overlay)")
	BuildCmd.Flags().BoolVarP(&Force, "force", "f", false, "Delete and overwrite an image if it currently exists")
	BuildCmd.Flags().BoolVarP(&NoTest, "notest", "T", false, "Bootstrap without running tests in %test section")
	BuildCmd.Flags().BoolVarP(&Remote, "remote", "r", false, "Build image remotely")
	BuildCmd.Flags().StringVar(&RemoteURL, "remote-url", "localhost:5050", "Specify the URL of the remote builder")
	BuildCmd.Flags().StringVar(&AuthToken, "auth-token", "", "Specify the auth token for the remote builder")
}

// BuildCmd represents the build command
var BuildCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args: cobra.ExactArgs(2),

	Use: docs.BuildUse,
	Short: docs.BuildShort,
	Long: docs.BuildLong,
	Example: docs.BuildExample,

	// TODO: Can we plz move this to another file to keep the CLI the CLI
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
				sylog.Fatalf("unable to parse URI %s: %v", args[1], err)
			}
		} else if !ok && err == nil {
			// Non-URI passed as arg[1]
			defFile, err := os.Open(args[1])
			if err != nil {
				sylog.Fatalf("unable to open file %s: %v", args[1], err)
			}
			defer defFile.Close()

			def, err = build.ParseDefinitionFile(defFile)
			if err != nil {
				sylog.Fatalf("failed to parse definition file %s: %v", args[1], err)
			}
		} else {
			sylog.Fatalf("unable to parse %s: %v", args[1], err)
		}

		if Remote {
			b = build.NewRemoteBuilder(args[0], def, false, RemoteURL, AuthToken)
		} else {
			b, err = build.NewSifBuilder(args[0], def)
			if err != nil {
				sylog.Fatalf("failed to create SifBuilder object: %v", err)
			}
		}

		if err := b.Build(context.TODO()); err != nil {
			sylog.Fatalf("failed to build image: %v", err)
		}
	},
	TraverseChildren: true,
}

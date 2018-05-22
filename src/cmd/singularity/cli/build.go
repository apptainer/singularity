// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/singularityware/singularity/src/pkg/build"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/spf13/cobra"
)

var (
	remote    bool
	remoteURL string
	authToken string
	sandbox   bool
	writable  bool
	force     bool
	noTest    bool
	sections  []string
)

func init() {
	buildCmd.Flags().SetInterspersed(false)
	singularityCmd.AddCommand(buildCmd)

	buildCmd.Flags().BoolVarP(&sandbox, "sandbox", "s", false, "Build image as sandbox format (chroot directory structure)")
	buildCmd.Flags().StringSliceVar(&sections, "section", []string{}, "Only run specific section(s) of deffile")
	buildCmd.Flags().BoolVarP(&writable, "writable", "w", false, "Build image as writable (SIF with writable internal overlay)")
	buildCmd.Flags().BoolVarP(&force, "force", "f", false, "")
	buildCmd.Flags().BoolVarP(&noTest, "notest", "T", false, "")
	buildCmd.Flags().BoolVarP(&remote, "remote", "r", false, "Build image remotely")
	buildCmd.Flags().StringVar(&remoteURL, "remote-url", "localhost:5050", "Specify the URL of the remote builder")
	buildCmd.Flags().StringVar(&authToken, "auth-token", "", "Specify the auth token for the remote builder")
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

		if sandbox {
			fmt.Println("Sandbox!")
		}

		if ok, err := build.IsValidURI(args[1]); ok && err == nil {
			// URI passed as arg[1]
			def, err = build.NewDefinitionFromURI(args[1])
			if err != nil {
				sylog.Fatalf("unable to parse URI %s: %v\n", args[1], err)
			}
		} else if !ok && err == nil {
			// Non-URI passed as arg[1]
			defFile, err := os.Open(args[1])
			if err != nil {
				sylog.Fatalf("unable to open file %s: %v\n", args[1], err)
			}
			defer defFile.Close()

			def, err = build.ParseDefinitionFile(defFile)
			if err != nil {
				sylog.Fatalf("failed to parse definition file %s: %v\n", args[1], err)
			}
		} else {
			sylog.Fatalf("unable to parse %s: %v\n", args[1], err)
		}

		if remote {
			b = build.NewRemoteBuilder(args[0], def, false, remoteURL, authToken)
		} else {
			b, err = build.NewSIFBuilder(args[0], def)
			if err != nil {
				sylog.Fatalf("failed to create SifBuilder object: %v\n", err)
			}
		}

		if err := b.Build(context.TODO()); err != nil {
			sylog.Fatalf("failed to build image: %v\n", err)
		}
	},
	TraverseChildren: true,
}

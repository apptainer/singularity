// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/build"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/auth"
	"github.com/spf13/cobra"
)

var (
	remote    bool
	remoteURL string
	json      bool
	sandbox   bool
	writable  bool
	force     bool
	noTest    bool
	sections  []string
)

func init() {
	BuildCmd.Flags().SetInterspersed(false)

	BuildCmd.Flags().BoolVarP(&sandbox, "sandbox", "s", false, "Build image as sandbox format (chroot directory structure)")
	BuildCmd.Flags().StringSliceVar(&sections, "section", []string{}, "Only run specific section(s) of deffile (setup, post, files, environment, test, labels, none)")
	BuildCmd.Flags().BoolVar(&json, "json", false, "Interpret build definition as JSON")
	BuildCmd.Flags().BoolVarP(&writable, "writable", "w", false, "Build image as writable (SIF with writable internal overlay)")
	BuildCmd.Flags().BoolVarP(&force, "force", "f", false, "Delete and overwrite an image if it currently exists")
	BuildCmd.Flags().BoolVarP(&noTest, "notest", "T", false, "Bootstrap without running tests in %test section")
	BuildCmd.Flags().BoolVarP(&remote, "remote", "r", false, "Build image remotely")
	BuildCmd.Flags().StringVar(&remoteURL, "remote-url", "localhost:5050", "Specify the URL of the remote builder")

	SingularityCmd.AddCommand(BuildCmd)
}

// BuildCmd represents the build command
var BuildCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args: cobra.ExactArgs(2),

	Use:     docs.BuildUse,
	Short:   docs.BuildShort,
	Long:    docs.BuildLong,
	Example: docs.BuildExample,
	PreRun:  sylabsToken,
	// TODO: Can we plz move this to another file to keep the CLI the CLI
	Run: func(cmd *cobra.Command, args []string) {
		var def build.Definition
		var err error

		var bundle *build.Bundle
		var cp build.ConveyorPacker

		if silent {
			fmt.Println("Silent!")
		}

		if sandbox {
			fmt.Println("Sandbox!")
		}

		if json {
			// b, err = build.NewSIFBuilderJSON(args[0], strings.NewReader(args[1]))
			// if err != nil {
			// 	sylog.Fatalf("Unable to parse JSON: %v\n", err)
			// }
			sylog.Fatalf("Build from JSON not implemented")
		} else {
			if ok, err := build.IsValidURI(args[1]); ok && err == nil {
				// URI passed as arg[1]
				// def, err = build.NewDefinitionFromURI(args[1])
				// if err != nil {
				// 	sylog.Fatalf("unable to parse URI %s: %v\n", args[1], err)
				// }

				cp = &build.DockerConveyorPacker{}

				u := strings.SplitN(args[1], ":", 2)

				if len(u) != 2 {
					return
				}

				if err = cp.Get(u[1]); err != nil {
					sylog.Fatalf("Conveyor failed to get:", err)
				}

				bundle, err = cp.Pack()
				if err != nil {
					sylog.Fatalf("Packer failed to pack:", err)
				}

			} else if !ok && err == nil {
				// // Non-URI passed as arg[1]
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
				var b *build.RemoteBuilder
				if authWarning != auth.WarningEmptyToken &&
					authWarning != auth.WarningTokenToolong &&
					authWarning != auth.WarningTokenTooShort {
					if authToken != "" {
						b = build.NewRemoteBuilder(args[0], def, false, remoteURL, authToken)
					}
				} else {
					sylog.Fatalf("Unable to submit build job: %v", authWarning)
				}

				if err := b.Build(context.TODO()); err != nil {
					sylog.Fatalf("failed to build image: %v\n", err)
				}

			} else {

				a := &build.SIFAssembler{}

				err = a.Assemble(bundle, args[0])
				if err != nil {
					sylog.Fatalf("Assembler failed to assemble:", err)
				}

				// b, err = build.NewSIFBuilder(args[0], def)
				// if err != nil {
				// 	sylog.Fatalf("failed to create SifBuilder object: %v\n", err)
				// }

			}
		}

	},
	TraverseChildren: true,
}

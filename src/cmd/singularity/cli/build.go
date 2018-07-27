// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/build"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/spf13/cobra"
)

var (
	remote     bool
	builderURL string
	detached   bool
	libraryURL string
	isJSON     bool
	sandbox    bool
	writable   bool
	force      bool
	noTest     bool
	sections   []string
)

const defbuilderURL = "localhost:5050"

func init() {
	BuildCmd.Flags().SetInterspersed(false)

	BuildCmd.Flags().BoolVarP(&sandbox, "sandbox", "s", false, "Build image as sandbox format (chroot directory structure)")
	BuildCmd.Flags().StringSliceVar(&sections, "section", []string{}, "Only run specific section(s) of deffile (setup, post, files, environment, test, labels, none)")
	BuildCmd.Flags().BoolVar(&isJSON, "json", false, "Interpret build definition as JSON")
	BuildCmd.Flags().BoolVarP(&writable, "writable", "w", false, "Build image as writable (SIF with writable internal overlay)")
	BuildCmd.Flags().BoolVarP(&force, "force", "f", false, "Delete and overwrite an image if it currently exists")
	BuildCmd.Flags().BoolVarP(&noTest, "notest", "T", false, "Bootstrap without running tests in %test section")
	BuildCmd.Flags().BoolVarP(&remote, "remote", "r", false, "Build image remotely")
	BuildCmd.Flags().BoolVarP(&detached, "detached", "d", false, "Submit build job and print nuild ID (no real-time logs)")
	BuildCmd.Flags().StringVar(&builderURL, "builder", defbuilderURL, "Specify the URL of the remote builder")
	BuildCmd.Flags().StringVar(&libraryURL, "library", "https://library.sylabs.io", "")

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
		var err error
		var a build.Assembler

		if silent {
			fmt.Println("Silent!")
		}

		if sandbox {
			fmt.Println("Sandbox!")
		}

		//check if target collides with existing file
		if ok := checkBuildTargetCollision(args[0], force); !ok {
			return
		}

		if remote || builderURL != defbuilderURL {
			// Submiting a remote build requires a valid authToken
			var b *build.RemoteBuilder
			if authToken != "" {
				b = build.NewRemoteBuilder(args[0], libraryURL, def, detached, builderURL, authToken)
			} else {
				sylog.Fatalf("Unable to submit build job: %v", authWarning)
			}
			b.Build(context.TODO())

		} else {
			//local build
			bundle := makeBundle(def)

			if syscall.Getuid() == 0 {
				doSections(bundle, args[0])
			} else if hasSections(def) {
				sylog.Warningf("Skipping definition scripts, not running as root [uid=%v]\n", syscall.Getuid())
			}

			if sandbox {
				a = &build.SandboxAssembler{}
			} else {
				a = &build.SIFAssembler{}
			}

			err = a.Assemble(bundle, args[0])
			if err != nil {
				sylog.Fatalf("Assembler failed to assemble: %v\n", err)
			}
		}

	},
	TraverseChildren: true,
}

// checkTargetCollision makes sure output target doesnt exist, or is ok to overwrite
func checkBuildTargetCollision(path string, force bool) bool {
	if _, err := os.Stat(path); err == nil {
		//exists
		if force {
			os.RemoveAll(path)
		} else {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Build target already exists. Do you want to overwrite? [N/y] ")
			input, err := reader.ReadString('\n')
			if err != nil {
				sylog.Fatalf("Error parsing input:", err)
			}
			if val := strings.Compare(strings.ToLower(input), "y\n"); val == 0 {
				os.RemoveAll(path)
			} else {
				fmt.Println("Stopping build.")
				return false
			}
		}
	}
	return true
}

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
	update     bool
	noTest     bool
	sections   []string
)

func init() {
	BuildCmd.Flags().SetInterspersed(false)

	BuildCmd.Flags().BoolVarP(&sandbox, "sandbox", "s", false, "Build image as sandbox format (chroot directory structure)")
	BuildCmd.Flags().StringSliceVar(&sections, "section", []string{"all"}, "Only run specific section(s) of deffile (setup, post, files, environment, test, labels, none)")
	BuildCmd.Flags().BoolVar(&isJSON, "json", false, "Interpret build definition as JSON")
	BuildCmd.Flags().BoolVarP(&writable, "writable", "w", false, "Build image as writable (SIF with writable internal overlay)")
	BuildCmd.Flags().BoolVarP(&force, "force", "F", false, "Delete and overwrite an image if it currently exists")
	BuildCmd.Flags().BoolVarP(&update, "update", "u", false, "Run definition over existing container")
	BuildCmd.Flags().BoolVarP(&noTest, "notest", "T", false, "Bootstrap without running tests in %test section")
	BuildCmd.Flags().BoolVarP(&remote, "remote", "r", false, "Build image remotely")
	BuildCmd.Flags().BoolVarP(&detached, "detached", "d", false, "Submit build job and print nuild ID (no real-time logs)")
	BuildCmd.Flags().StringVar(&builderURL, "builder", "https://build.sylabs.io", "Remote Build Service URL")
	BuildCmd.Flags().StringVar(&libraryURL, "library", "https://library.sylabs.io", "Container Library URL")

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
		buildFormat := "sif"
		if sandbox {
			buildFormat = "sandbox"
		}

		dest := args[0]
		spec := args[1]

		//check if target collides with existing file
		if ok := checkBuildTarget(dest); !ok {
			os.Exit(1)
		}

		if remote {
			// Submiting a remote build requires a valid authToken
			if authToken == "" {
				sylog.Fatalf("Unable to submit build job: %v", authWarning)
			}

			def, err := build.MakeDef(spec)
			if err != nil {
				return
			}

			b, err := build.NewRemoteBuilder(dest, libraryURL, def, detached, builderURL, authToken)
			if err != nil {
				sylog.Fatalf("failed to create builder: %v", err)
			}
			b.Build(context.TODO())
		} else {

			err := checkSections()
			if err != nil {
				sylog.Fatalf(err.Error())
			}

			b, err := build.NewBuild(spec, dest, buildFormat, force, update, sections, noTest)
			if err != nil {
				sylog.Fatalf("Unable to create build: %v", err)
			}

			if err = b.Full(); err != nil {
				sylog.Fatalf("While performing build: %v", err)
			}
		}
	},
	TraverseChildren: true,
}

// checkTargetCollision makes sure output target doesnt exist, or is ok to overwrite
func checkBuildTarget(path string) bool {
	if f, err := os.Stat(path); err == nil {
		if update && !f.IsDir() {
			sylog.Fatalf("Only sandbox updating is supported.")
		}
		if !update && !force {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Build target already exists. Do you want to overwrite? [N/y] ")
			input, err := reader.ReadString('\n')
			if err != nil {
				sylog.Fatalf("Error parsing input: %s", err)
			}
			if val := strings.Compare(strings.ToLower(input), "y\n"); val == 0 {
				force = true
			} else {
				sylog.Errorf("Stopping build.")
				return false
			}
		}
	}
	return true
}

func checkSections() error {
	var all, none bool
	for _, section := range sections {
		if section == "none" {
			none = true
		}
		if section == "all" {
			all = true
		}
	}

	if all && len(sections) > 1 {
		return fmt.Errorf("Section specification error: Cannot have all and any other option")
	}
	if none && len(sections) > 1 {
		return fmt.Errorf("Section specification error: Cannot have none and any other option")
	}

	return nil
}

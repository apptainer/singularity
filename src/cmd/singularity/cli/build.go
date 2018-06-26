// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/build"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
	"github.com/singularityware/singularity/src/runtime/engines/common/oci"
	"github.com/singularityware/singularity/src/runtime/engines/imgbuild"
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

		def := makeDefinition(args[1], isJSON)

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
			doSections(bundle, args[0])

			if sandbox {
				sylog.Fatalf("Cannot build to sandbox... yet\n")
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

func doSections(b *build.Bundle, fullPath string) {
	lvl := "0"
	if verbose {
		lvl = "2"
	}
	if debug {
		lvl = "5"
	}

	wrapper := filepath.Join(buildcfg.SBINDIR, "/wrapper")
	progname := []string{"singularity image-build"}
	env := []string{"SINGULARITY_MESSAGELEVEL=" + lvl, "SRUNTIME=imgbuild"}

	engineConfig := &imgbuild.EngineConfig{
		Bundle: *b,
	}
	ociConfig := &oci.Config{}

	config := &config.Common{
		EngineName:   imgbuild.Name,
		ContainerID:  filepath.Base(fullPath),
		OciConfig:    ociConfig,
		EngineConfig: engineConfig,
	}

	configData, err := json.Marshal(config)
	if err != nil {
		sylog.Fatalf("CLI Failed to marshal CommonEngineConfig: %s\n", err)
	}

	// Create os/exec.Command to run wrapper and return control once finished
	wrapperCmd := &exec.Cmd{
		Path:   wrapper,
		Args:   progname,
		Env:    env,
		Stdin:  bytes.NewReader(configData),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := wrapperCmd.Start(); err != nil {
		sylog.Fatalf("Unable to start wrapper proc: %v\n", err)
	}
	if err := wrapperCmd.Wait(); err != nil {
		sylog.Fatalf("wrapper proc: %v\n", err)
	}
}

// getDefinition creates definition object from various input sources
func makeDefinition(source string, isJSON bool) build.Definition {
	var err error
	var def build.Definition

	if isJSON {
		def, err = build.NewDefinitionFromJSON(strings.NewReader(source))
		if err != nil {
			sylog.Fatalf("Unable to parse JSON: %v\n", err)
		}

	} else if ok, err := build.IsValidURI(source); ok && err == nil {
		// URI passed as arg[1]
		def, err = build.NewDefinitionFromURI(source)
		if err != nil {
			sylog.Fatalf("unable to parse URI %s: %v\n", source, err)
		}

	} else if !ok && err == nil {
		// Non-URI passed as arg[1]
		defFile, err := os.Open(source)
		if err != nil {
			sylog.Fatalf("unable to open file %s: %v\n", source, err)
		}
		defer defFile.Close()

		def, err = build.ParseDefinitionFile(defFile)
		if err != nil {
			sylog.Fatalf("failed to parse definition file %s: %v\n", source, err)
		}

	} else {
		sylog.Fatalf("unable to parse %s: %v\n", source, err)
	}

	return def
}

// makeBundle creates a bundle by Getting and Packing from the proper source
func makeBundle(def build.Definition) *build.Bundle {
	var err error
	var cp build.ConveyorPacker

	switch def.Header["bootstrap"] {
	case "docker":
		cp = &build.DockerConveyorPacker{}
	default:
		sylog.Fatalf("Not a valid build source %s: %v\n", def.Header["bootstrap"], err)
	}

	if err = cp.Get(def); err != nil {
		sylog.Fatalf("Conveyor failed to get: %v\n", err)
	}

	bundle, err := cp.Pack()
	if err != nil {
		sylog.Fatalf("Packer failed to pack: %v\n", err)
	}

	return bundle
}

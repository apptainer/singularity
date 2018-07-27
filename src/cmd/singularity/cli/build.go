// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/build"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/cli"
	syexec "github.com/singularityware/singularity/src/pkg/util/exec"
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
	// TODO: Can we plz move this to another file to keep the CLI the CLI
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var a build.Assembler

		// get the Sylabs token
		authToken, authWarning := cli.SylabsToken(defaultTokenFile, tokenFile)

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

// hasSections returns true if build definition is requesting to run scripts in image
func hasSections(def build.Definition) bool {
	return def.BuildData.Post != "" || def.BuildData.Pre != "" || def.BuildData.Setup != ""
}

// doSections invokes the imgbuild engine through wrapper
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

	// Set PIPE_EXEC_FD
	pipefd, err := syexec.SetPipe(configData)
	if err != nil {
		sylog.Fatalf("Failed to set PIPE_EXEC_FD: %v\n", err)
	}

	env = append(env, pipefd)

	// Create os/exec.Command to run wrapper and return control once finished
	wrapperCmd := &exec.Cmd{
		Path:   wrapper,
		Args:   progname,
		Env:    env,
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

	} else if ok, err := build.IsValidDefinition(source); ok && err == nil {
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
	} else if _, err := os.Stat(source); err == nil {
		//local image or sandbox
		def = build.Definition{
			Header: map[string]string{
				"bootstrap": "localimage",
				"from":      source,
			},
		}
	} else {
		sylog.Fatalf("unable to build from %s: %v\n", source, err)
	}

	return def
}

// makeBundle creates a bundle by Getting and Packing from the proper source
func makeBundle(def build.Definition) *build.Bundle {
	var err error
	var cp build.ConveyorPacker

	switch def.Header["bootstrap"] {
	case "shub":
		cp = &build.ShubConveyorPacker{}
	case "docker", "docker-archive", "docker-daemon", "oci", "oci-archive":
		cp = &build.OCIConveyorPacker{}
	case "busybox":
		cp = &build.BusyBoxConveyorPacker{}
	case "debootstrap":
		cp = &build.DebootstrapConveyorPacker{}
	case "arch":
		cp = &build.ArchConveyorPacker{}
	case "localimage":
		cp = &build.LocalConveyorPacker{}
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

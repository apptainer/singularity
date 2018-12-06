// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/sylabs/singularity/internal/pkg/build/types"
	"github.com/sylabs/singularity/internal/pkg/build/types/parser"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/src/docs"
)

var (
	remote         bool
	builderURL     string
	detached       bool
	libraryURL     string
	isJSON         bool
	sandbox        bool
	writable       bool
	force          bool
	update         bool
	noTest         bool
	sections       []string
	noHTTPS        bool
	tmpDir         string
	dockerUsername string
	dockerPassword string
	dockerLogin    bool
)

var buildflags = pflag.NewFlagSet("BuildFlags", pflag.ExitOnError)

func init() {
	BuildCmd.Flags().SetInterspersed(false)

	BuildCmd.Flags().BoolVarP(&sandbox, "sandbox", "s", false, "build image as sandbox format (chroot directory structure)")
	BuildCmd.Flags().SetAnnotation("sandbox", "envkey", []string{"SANDBOX"})

	BuildCmd.Flags().StringSliceVar(&sections, "section", []string{"all"}, "only run specific section(s) of deffile (setup, post, files, environment, test, labels, none)")
	BuildCmd.Flags().SetAnnotation("section", "envkey", []string{"SECTION"})

	BuildCmd.Flags().BoolVar(&isJSON, "json", false, "interpret build definition as JSON")
	BuildCmd.Flags().SetAnnotation("json", "envkey", []string{"JSON"})

	BuildCmd.Flags().BoolVarP(&force, "force", "F", false, "delete and overwrite an image if it currently exists")
	BuildCmd.Flags().SetAnnotation("force", "envkey", []string{"FORCE"})

	BuildCmd.Flags().BoolVarP(&update, "update", "u", false, "run definition over existing container (skips header)")
	BuildCmd.Flags().SetAnnotation("update", "envkey", []string{"UPDATE"})

	BuildCmd.Flags().BoolVarP(&noTest, "notest", "T", false, "build without running tests in %test section")
	BuildCmd.Flags().SetAnnotation("notest", "envkey", []string{"NOTEST"})

	BuildCmd.Flags().BoolVarP(&remote, "remote", "r", false, "build image remotely (does not require root)")
	BuildCmd.Flags().SetAnnotation("remote", "envkey", []string{"REMOTE"})

	BuildCmd.Flags().BoolVarP(&detached, "detached", "d", false, "submit build job and print nuild ID (no real-time logs and requires --remote)")
	BuildCmd.Flags().SetAnnotation("detached", "envkey", []string{"DETACHED"})

	BuildCmd.Flags().StringVar(&builderURL, "builder", "https://build.sylabs.io", "remote Build Service URL")
	BuildCmd.Flags().SetAnnotation("builder", "envkey", []string{"BUILDER"})

	BuildCmd.Flags().StringVar(&libraryURL, "library", "https://library.sylabs.io", "container Library URL")
	BuildCmd.Flags().SetAnnotation("library", "envkey", []string{"LIBRARY"})

	BuildCmd.Flags().StringVar(&tmpDir, "tmpdir", "", "specify a temporary directory to use for build")
	BuildCmd.Flags().SetAnnotation("tmpdir", "envkey", []string{"TMPDIR"})

	BuildCmd.Flags().BoolVar(&noHTTPS, "nohttps", false, "do NOT use HTTPS, for communicating with local docker registry")
	BuildCmd.Flags().SetAnnotation("nohttps", "envkey", []string{"NOHTTPS"})

	BuildCmd.Flags().StringVar(&dockerUsername, "docker-username", "", "specify a username for docker authentication")
	BuildCmd.Flags().Lookup("docker-username").Hidden = true
	BuildCmd.Flags().SetAnnotation("docker-username", "envkey", []string{"DOCKER_USERNAME"})

	BuildCmd.Flags().StringVar(&dockerPassword, "docker-password", "", "specify a password for docker authentication")
	BuildCmd.Flags().Lookup("docker-password").Hidden = true
	BuildCmd.Flags().SetAnnotation("docker-password", "envkey", []string{"DOCKER_PASSWORD"})

	BuildCmd.Flags().BoolVar(&dockerLogin, "docker-login", false, "interactive prompt for docker authentication")
	BuildCmd.Flags().SetAnnotation("docker-login", "envkey", []string{"DOCKER_LOGIN"})

	SingularityCmd.AddCommand(BuildCmd)
}

// BuildCmd represents the build command
var BuildCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(2),

	Use:              docs.BuildUse,
	Short:            docs.BuildShort,
	Long:             docs.BuildLong,
	Example:          docs.BuildExample,
	PreRun:           preRun,
	Run:              run,
	TraverseChildren: true,
}

// checkTargetCollision makes sure output target doesn't exist, or is ok to overwrite
func checkBuildTarget(path string, update bool) bool {
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

func definitionFromSpec(spec string) (def types.Definition, err error) {

	// Try spec as URI first
	def, err = types.NewDefinitionFromURI(spec)
	if err == nil {
		return
	}

	// Try spec as local file
	var isValid bool
	isValid, err = parser.IsValidDefinition(spec)
	if err != nil {
		return
	}

	if isValid {
		sylog.Debugf("Found valid definition: %s\n", spec)
		// File exists and contains valid definition
		var defFile *os.File
		defFile, err = os.Open(spec)
		if err != nil {
			return
		}

		defer defFile.Close()
		def, err = parser.ParseDefinitionFile(defFile)

		return
	}

	// File exists and does NOT contain a valid definition
	// local image or sandbox
	def = types.Definition{
		Header: map[string]string{
			"bootstrap": "localimage",
			"from":      spec,
		},
	}

	return
}

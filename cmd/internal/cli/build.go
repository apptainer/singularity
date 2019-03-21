// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	ocitypes "github.com/containers/image/types"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/build/types/parser"
	"github.com/sylabs/singularity/pkg/sypgp"
)

var (
	remote         bool
	builderURL     string
	detached       bool
	libraryURL     string
	isJSON         bool
	sandbox        bool
	force          bool
	update         bool
	noTest         bool
	sections       []string
	noHTTPS        bool
	tmpDir         string
	dockerUsername string
	dockerPassword string
	dockerLogin    bool
	noCleanUp      bool
)

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

	BuildCmd.Flags().BoolVarP(&detached, "detached", "d", false, "submit build job and print build ID (no real-time logs and requires --remote)")
	BuildCmd.Flags().SetAnnotation("detached", "envkey", []string{"DETACHED"})

	BuildCmd.Flags().StringVar(&builderURL, "builder", "https://build.sylabs.io", "remote Build Service URL, setting this implies --remote")
	BuildCmd.Flags().SetAnnotation("builder", "envkey", []string{"BUILDER"})

	BuildCmd.Flags().StringVar(&libraryURL, "library", "https://library.sylabs.io", "container Library URL")
	BuildCmd.Flags().SetAnnotation("library", "envkey", []string{"LIBRARY"})

	BuildCmd.Flags().StringVar(&tmpDir, "tmpdir", "", "specify a temporary directory to use for build")
	BuildCmd.Flags().SetAnnotation("tmpdir", "envkey", []string{"TMPDIR"})

	BuildCmd.Flags().BoolVar(&noHTTPS, "nohttps", false, "do NOT use HTTPS, for communicating with local docker registry")
	BuildCmd.Flags().SetAnnotation("nohttps", "envkey", []string{"NOHTTPS"})

	BuildCmd.Flags().BoolVar(&noCleanUp, "no-cleanup", false, "do NOT clean up bundle after failed build, can be helpul for debugging")
	BuildCmd.Flags().SetAnnotation("no-cleanup", "envkey", []string{"NO_CLEANUP"})

	BuildCmd.Flags().AddFlag(actionFlags.Lookup("docker-username"))
	BuildCmd.Flags().AddFlag(actionFlags.Lookup("docker-password"))
	BuildCmd.Flags().AddFlag(actionFlags.Lookup("docker-login"))

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

func preRun(cmd *cobra.Command, args []string) {
	// Always perform remote build when builder flag is set
	if cmd.Flags().Lookup("builder").Changed {
		cmd.Flags().Lookup("remote").Value.Set("true")
	}

	sylabsToken(cmd, args)
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

func makeDockerCredentials(cmd *cobra.Command) (authConf *ocitypes.DockerAuthConfig, err error) {
	usernameFlag := cmd.Flags().Lookup("docker-username")
	passwordFlag := cmd.Flags().Lookup("docker-password")

	if dockerLogin {
		if !usernameFlag.Changed {
			dockerUsername, err = sypgp.AskQuestion("Enter Docker Username: ")
			if err != nil {
				return
			}
			usernameFlag.Value.Set(dockerUsername)
			usernameFlag.Changed = true
		}

		dockerPassword, err = sypgp.AskQuestionNoEcho("Enter Docker Password: ")
		if err != nil {
			return
		}
		passwordFlag.Value.Set(dockerPassword)
		passwordFlag.Changed = true
	}

	if usernameFlag.Changed && passwordFlag.Changed {
		authConf = &ocitypes.DockerAuthConfig{
			Username: dockerUsername,
			Password: dockerPassword,
		}
	}

	return
}

// remote builds need to fail if we cannot resolve remote URLS
func handleRemoteBuildFlags(cmd *cobra.Command) {
	// if we can load config and if default endpoint is set, use that
	// otherwise fall back on regular authtoken and URI behavior
	endpoint, err := sylabsRemote(remoteConfig)
	if err == scs.ErrNoDefault {
		sylog.Warningf("No default remote in use, falling back to CLI defaults")
		return
	} else if err != nil {
		sylog.Fatalf("Unable to load remote configuration: %v", err)
	}

	authToken = endpoint.Token
	if !cmd.Flags().Lookup("builder").Changed {
		uri, err := endpoint.GetServiceURI("builder")
		if err != nil {
			sylog.Fatalf("Unable to get build service URI: %v", err)
		}
		builderURL = uri
	}
	if !cmd.Flags().Lookup("library").Changed {
		uri, err := endpoint.GetServiceURI("library")
		if err != nil {
			sylog.Fatalf("Unable to get library service URI: %v", err)
		}
		libraryURL = uri
	}
}

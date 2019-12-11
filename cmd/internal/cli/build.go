// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"
	"runtime"

	ocitypes "github.com/containers/image/types"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/interactive"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/build/types/parser"
	"github.com/sylabs/singularity/pkg/cmdline"
)

var buildArgs struct {
	sections   []string
	arch       string
	builderURL string
	libraryURL string
	detached   bool
	encrypt    bool
	fakeroot   bool
	fixPerms   bool
	isJSON     bool
	noCleanUp  bool
	noTest     bool
	remote     bool
	sandbox    bool
	update     bool
}

// -s|--sandbox
var buildSandboxFlag = cmdline.Flag{
	ID:           "buildSandboxFlag",
	Value:        &buildArgs.sandbox,
	DefaultValue: false,
	Name:         "sandbox",
	ShortHand:    "s",
	Usage:        "build image as sandbox format (chroot directory structure)",
	EnvKeys:      []string{"SANDBOX"},
}

// --section
var buildSectionFlag = cmdline.Flag{
	ID:           "buildSectionFlag",
	Value:        &buildArgs.sections,
	DefaultValue: []string{"all"},
	Name:         "section",
	Usage:        "only run specific section(s) of deffile (setup, post, files, environment, test, labels, none)",
	EnvKeys:      []string{"SECTION"},
}

// --json
var buildJSONFlag = cmdline.Flag{
	ID:           "buildJSONFlag",
	Value:        &buildArgs.isJSON,
	DefaultValue: false,
	Name:         "json",
	Usage:        "interpret build definition as JSON",
	EnvKeys:      []string{"JSON"},
}

// -u|--update
var buildUpdateFlag = cmdline.Flag{
	ID:           "buildUpdateFlag",
	Value:        &buildArgs.update,
	DefaultValue: false,
	Name:         "update",
	ShortHand:    "u",
	Usage:        "run definition over existing container (skips header)",
	EnvKeys:      []string{"UPDATE"},
}

// -T|--notest
var buildNoTestFlag = cmdline.Flag{
	ID:           "buildNoTestFlag",
	Value:        &buildArgs.noTest,
	DefaultValue: false,
	Name:         "notest",
	ShortHand:    "T",
	Usage:        "build without running tests in %test section",
	EnvKeys:      []string{"NOTEST"},
}

// -r|--remote
var buildRemoteFlag = cmdline.Flag{
	ID:           "buildRemoteFlag",
	Value:        &buildArgs.remote,
	DefaultValue: false,
	Name:         "remote",
	ShortHand:    "r",
	Usage:        "build image remotely (does not require root)",
	EnvKeys:      []string{"REMOTE"},
}

// --arch
var buildArchFlag = cmdline.Flag{
	ID:           "buildArchFlag",
	Value:        &buildArgs.arch,
	DefaultValue: runtime.GOARCH,
	Name:         "arch",
	Usage:        "architecture for remote build",
	EnvKeys:      []string{"BUILD_ARCH"},
}

// -d|--detached
var buildDetachedFlag = cmdline.Flag{
	ID:           "buildDetachedFlag",
	Value:        &buildArgs.detached,
	DefaultValue: false,
	Name:         "detached",
	ShortHand:    "d",
	Usage:        "submit build job and print build ID (no real-time logs and requires --remote)",
	EnvKeys:      []string{"DETACHED"},
}

// --builder
var buildBuilderFlag = cmdline.Flag{
	ID:           "buildBuilderFlag",
	Value:        &buildArgs.builderURL,
	DefaultValue: "https://build.sylabs.io",
	Name:         "builder",
	Usage:        "remote Build Service URL, setting this implies --remote",
	EnvKeys:      []string{"BUILDER"},
}

// --library
var buildLibraryFlag = cmdline.Flag{
	ID:           "buildLibraryFlag",
	Value:        &buildArgs.libraryURL,
	DefaultValue: "https://library.sylabs.io",
	Name:         "library",
	Usage:        "container Library URL",
	EnvKeys:      []string{"LIBRARY"},
}

// --disable-cache
var buildDisableCacheFlag = cmdline.Flag{
	ID:           "buildDisableCacheFlag",
	Value:        &disableCache,
	DefaultValue: false,
	Name:         "disable-cache",
	Usage:        "do not use cache or create cache",
	EnvKeys:      []string{"DISABLE_CACHE"},
}

// --no-cleanup
var buildNoCleanupFlag = cmdline.Flag{
	ID:           "buildNoCleanupFlag",
	Value:        &buildArgs.noCleanUp,
	DefaultValue: false,
	Name:         "no-cleanup",
	Usage:        "do NOT clean up bundle after failed build, can be helpul for debugging",
	EnvKeys:      []string{"NO_CLEANUP"},
}

// --fakeroot
var buildFakerootFlag = cmdline.Flag{
	ID:           "buildFakerootFlag",
	Value:        &buildArgs.fakeroot,
	DefaultValue: false,
	Name:         "fakeroot",
	ShortHand:    "f",
	Usage:        "build using user namespace to fake root user (requires a privileged installation)",
	EnvKeys:      []string{"FAKEROOT"},
}

// -e|--encrypt
var buildEncryptFlag = cmdline.Flag{
	ID:           "buildEncryptFlag",
	Value:        &buildArgs.encrypt,
	DefaultValue: false,
	Name:         "encrypt",
	ShortHand:    "e",
	Usage:        "build an image with an encrypted file system",
}

// TODO: Deprecate at 3.6, remove at 3.8
// --fix-perms
var buildFixPermsFlag = cmdline.Flag{
	ID:           "fixPermsFlag",
	Value:        &buildArgs.fixPerms,
	DefaultValue: false,
	Name:         "fix-perms",
	Usage:        "ensure owner has rwX permissions on all container content for oci/docker sources",
	EnvKeys:      []string{"FIXPERMS"},
}

func init() {
	cmdManager.RegisterCmd(buildCmd)

	cmdManager.RegisterFlagForCmd(&buildArchFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&buildBuilderFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&buildDetachedFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&buildDisableCacheFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&buildEncryptFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&buildFakerootFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&buildFixPermsFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&buildJSONFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&buildLibraryFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&buildNoCleanupFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&buildNoTestFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&buildRemoteFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&buildSandboxFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&buildSectionFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&buildUpdateFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&commonForceFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&commonNoHTTPSFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&commonTmpDirFlag, buildCmd)

	cmdManager.RegisterFlagForCmd(&dockerUsernameFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&dockerPasswordFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&dockerLoginFlag, buildCmd)

	cmdManager.RegisterFlagForCmd(&commonPromptForPassphraseFlag, buildCmd)
	cmdManager.RegisterFlagForCmd(&commonPEMFlag, buildCmd)
}

// buildCmd represents the build command.
var buildCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(2),

	Use:              docs.BuildUse,
	Short:            docs.BuildShort,
	Long:             docs.BuildLong,
	Example:          docs.BuildExample,
	PreRun:           preRun,
	Run:              runBuild,
	TraverseChildren: true,
}

func preRun(cmd *cobra.Command, args []string) {
	if buildArgs.fakeroot && !buildArgs.remote {
		fakerootExec(args)
	}

	// Always perform remote build when builder flag is set
	if cmd.Flags().Lookup("builder").Changed {
		cmd.Flags().Lookup("remote").Value.Set("true")
	}

	sylabsToken(cmd, args)
}

// checkBuildTarget makes sure output target doesn't exist, or is ok to overwrite.
// And checks that update flag will update an existing directory.
func checkBuildTarget(path string) error {
	if !buildArgs.sandbox && buildArgs.update {
		return fmt.Errorf("only sandbox update is supported: --sandbox flag is missing")
	}
	if f, err := os.Stat(path); err == nil {
		if buildArgs.update && !f.IsDir() {
			return fmt.Errorf("only sandbox update is supported: %s is not a directory", path)
		}
		if !buildArgs.update && !forceOverwrite {
			question := "Build target already exists. Do you want to overwrite? [N/y] "
			input, err := interactive.AskYNQuestion("n", question)
			if err != nil {
				return fmt.Errorf("while reading the input: %s", err)
			}
			if input != "y" {
				return fmt.Errorf("stopping build")
			}
			forceOverwrite = true
		}
	} else if os.IsNotExist(err) && buildArgs.update && buildArgs.sandbox {
		return fmt.Errorf("could not update sandbox %s: doesn't exist", path)
	}
	return nil
}

// definitionFromSpec is specifically for parsing specs for the remote builder
// it uses a different version the the definition struct and parser
func definitionFromSpec(spec string) (types.Definition, error) {
	// Try spec as URI first
	def, err := types.NewDefinitionFromURI(spec)
	if err == nil {
		return def, nil
	}

	// Try spec as local file
	var isValid bool
	isValid, err = parser.IsValidDefinition(spec)
	if err != nil {
		return types.Definition{}, err
	}

	if isValid {
		sylog.Debugf("Found valid definition: %s\n", spec)
		// File exists and contains valid definition
		var defFile *os.File
		defFile, err = os.Open(spec)
		if err != nil {
			return types.Definition{}, err
		}

		defer defFile.Close()

		return parser.ParseDefinitionFile(defFile)
	}

	// File exists and does NOT contain a valid definition
	// local image or sandbox
	def = types.Definition{
		Header: map[string]string{
			"bootstrap": "localimage",
			"from":      spec,
		},
	}

	return def, nil
}

// makeDockerCredentials creates an *ocitypes.DockerAuthConfig to use for
// OCI/Docker registry operation configuration. Note that if we don't have a
// username or password set it will return a nil pointer, as containers/image
// requires this to fall back to .docker/config based authentication.
func makeDockerCredentials(cmd *cobra.Command) (authConf *ocitypes.DockerAuthConfig, err error) {
	usernameFlag := cmd.Flags().Lookup("docker-username")
	passwordFlag := cmd.Flags().Lookup("docker-password")

	if dockerLogin {
		if !usernameFlag.Changed {
			dockerAuthConfig.Username, err = interactive.AskQuestion("Enter Docker Username: ")
			if err != nil {
				return authConf, err
			}
			usernameFlag.Value.Set(dockerAuthConfig.Username)
			usernameFlag.Changed = true
		}

		dockerAuthConfig.Password, err = interactive.AskQuestionNoEcho("Enter Docker Password: ")
		if err != nil {
			return authConf, err
		}
		passwordFlag.Value.Set(dockerAuthConfig.Password)
		passwordFlag.Changed = true
	}

	if usernameFlag.Changed || passwordFlag.Changed {
		return &dockerAuthConfig, nil
	}

	// If a username / password have not been explicitly set, return a nil
	// pointer, which will mean containers/image falls back to looking for
	// .docker/config.json
	return nil, nil
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
		buildArgs.builderURL = uri
	}
	if !cmd.Flags().Lookup("library").Changed {
		uri, err := endpoint.GetServiceURI("library")
		if err != nil {
			sylog.Fatalf("Unable to get library service URI: %v", err)
		}
		buildArgs.libraryURL = uri
	}
}

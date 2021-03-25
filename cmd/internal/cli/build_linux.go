// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	osExec "os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/remote/endpoint"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/build/remotebuilder"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/cache"
	fakerootConfig "github.com/sylabs/singularity/internal/pkg/runtime/engine/fakeroot/config"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/interactive"
	"github.com/sylabs/singularity/internal/pkg/util/starter"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/runtime/engine/config"
	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/sylabs/singularity/pkg/util/crypt"
)

func fakerootExec(cmdArgs []string) {
	useSuid := buildcfg.SINGULARITY_SUID_INSTALL == 1

	short := "-" + buildFakerootFlag.ShortHand
	long := "--" + buildFakerootFlag.Name
	envKey := fmt.Sprintf("SINGULARITY_%s", buildFakerootFlag.EnvKeys[0])
	fakerootEnv := os.Getenv(envKey) != ""

	argsLen := len(os.Args) - 1
	if fakerootEnv {
		argsLen = len(os.Args)
		os.Unsetenv(envKey)
	}
	args := make([]string, argsLen)
	idx := 0
	for i, arg := range os.Args {
		if i == 0 {
			path, _ := osExec.LookPath(arg)
			arg = path
		}
		if arg != short && arg != long {
			args[idx] = arg
			idx++
		}

	}

	user, err := user.GetPwUID(uint32(os.Getuid()))
	if err != nil {
		sylog.Fatalf("failed to retrieve user information: %s", err)
	}

	// Append the user's real UID to the environment as _CONTAINERS_ROOTLESS_UID.
	// This is required in fakeroot builds that may use containers/image 5.7 and above.
	// https://github.com/containers/image/issues/1066
	// https://github.com/containers/image/blob/master/internal/rootless/rootless.go
	os.Setenv("_CONTAINERS_ROOTLESS_UID", strconv.Itoa(os.Getuid()))

	engineConfig := &fakerootConfig.EngineConfig{
		Args:     args,
		Envs:     os.Environ(),
		Home:     user.Dir,
		BuildEnv: true,
	}

	cfg := &config.Common{
		EngineName:   fakerootConfig.Name,
		ContainerID:  "fakeroot",
		EngineConfig: engineConfig,
	}

	err = starter.Exec(
		"Singularity fakeroot",
		cfg,
		starter.UseSuid(useSuid),
	)
	sylog.Fatalf("%s", err)
}

func runBuild(cmd *cobra.Command, args []string) {
	ctx := context.TODO()

	if buildArgs.arch != runtime.GOARCH && !buildArgs.remote {
		sylog.Fatalf("Requested architecture (%s) does not match host (%s). Cannot build locally.", buildArgs.arch, runtime.GOARCH)
	}

	dest := args[0]
	spec := args[1]

	// check if target collides with existing file
	if err := checkBuildTarget(dest); err != nil {
		sylog.Fatalf("While checking build target: %s", err)
	}

	if buildArgs.remote {
		runBuildRemote(ctx, cmd, dest, spec)
	} else {
		runBuildLocal(ctx, cmd, dest, spec)
	}
	sylog.Infof("Build complete: %s", dest)
}

func runBuildRemote(ctx context.Context, cmd *cobra.Command, dst, spec string) {
	// building encrypted containers on the remote builder is not currently supported
	if buildArgs.encrypt {
		sylog.Fatalf("Building encrypted container with the remote builder is not currently supported.")
	}

	// TODO - the keyserver config needs to go to the remote builder for fingerprint verification at
	// build time to be fully supported.
	bc, lc, _, err := getServiceConfigs(buildArgs.builderURL, buildArgs.libraryURL, buildArgs.keyServerURL)
	if err != nil {
		sylog.Fatalf("Unable to get builder and library client configuration: %v", err)
	}
	buildArgs.libraryURL = lc.BaseURL
	buildArgs.builderURL = bc.BaseURL

	// To provide a web link to detached remote builds we need to know the web frontend URI.
	// We only know this working forward from a remote config, and not if the user has set custom
	// service URLs, since there is no straightforward foolproof way to work back from them to a
	// matching frontend URL.
	if !cmd.Flag("builder").Changed && !cmd.Flag("library").Changed {
		buildArgs.webURL = URI()
	}

	// submitting a remote build requires a valid authToken
	if bc.AuthToken == "" {
		sylog.Fatalf("Unable to submit build job: %v", remoteWarning)
	}

	def, err := definitionFromSpec(spec)
	if err != nil {
		sylog.Fatalf("Unable to build from %s: %v", spec, err)
	}

	// path SIF from remote builder should be placed
	rbDst := dst
	if buildArgs.sandbox {
		if strings.HasPrefix(dst, "library://") {
			// image destination is the library.
			sylog.Fatalf("Library URI detected as destination, sandbox builds are incompatible with library destinations.")
		}

		// create temporary file to download sif
		f, err := ioutil.TempFile(tmpDir, "remote-build-")
		if err != nil {
			sylog.Fatalf("Could not create temporary directory: %s", err)
		}
		f.Close()

		// override remote build destation to temporary file for conversion to a sandbox
		rbDst = f.Name()
		sylog.Debugf("Overriding remote build destination to temporary file: %s", rbDst)

		// remove downloaded sif
		defer os.Remove(rbDst)

		// build from sif downloaded in tmp location
		defer func() {
			sylog.Debugf("Building sandbox from downloaded SIF")
			imgCache := getCacheHandle(cache.Config{Disable: disableCache})
			if imgCache == nil {
				sylog.Fatalf("failed to create an image cache handle")
			}

			d, err := types.NewDefinitionFromURI("localimage" + "://" + rbDst)
			if err != nil {
				sylog.Fatalf("Unable to create definition for sandbox build: %v", err)
			}

			b, err := build.New(
				[]types.Definition{d},
				build.Config{
					Dest:      dst,
					Format:    "sandbox",
					NoCleanUp: buildArgs.noCleanUp,
					Opts: types.Options{
						ImgCache: imgCache,
						NoCache:  disableCache,
						TmpDir:   tmpDir,
						Update:   buildArgs.update,
						Force:    forceOverwrite,
					},
				})
			if err != nil {
				sylog.Fatalf("Unable to create build: %v", err)
			}

			if err = b.Full(ctx); err != nil {
				sylog.Fatalf("While performing build: %v", err)
			}
		}()
	}

	b, err := remotebuilder.New(rbDst, buildArgs.libraryURL, def, buildArgs.detached, forceOverwrite, buildArgs.builderURL, bc.AuthToken, buildArgs.arch, buildArgs.webURL)
	if err != nil {
		sylog.Fatalf("Failed to create builder: %v", err)
	}
	err = b.Build(ctx)
	if err != nil {
		sylog.Fatalf("While performing build: %v", err)
	}
}

func runBuildLocal(ctx context.Context, cmd *cobra.Command, dst, spec string) {
	var keyInfo *crypt.KeyInfo
	if buildArgs.encrypt || promptForPassphrase || cmd.Flags().Lookup("pem-path").Changed {
		if os.Getuid() != 0 {
			sylog.Fatalf("You must be root to build an encrypted container")
		}

		k, err := getEncryptionMaterial(cmd)
		if err != nil {
			sylog.Fatalf("While handling encryption material: %v", err)
		}
		keyInfo = &k
	} else {
		_, passphraseEnvOK := os.LookupEnv("SINGULARITY_ENCRYPTION_PASSPHRASE")
		_, pemPathEnvOK := os.LookupEnv("SINGULARITY_ENCRYPTION_PEM_PATH")
		if passphraseEnvOK || pemPathEnvOK {
			sylog.Warningf("Encryption related env vars found, but --encrypt was not specified. NOT encrypting container.")
		}
	}

	imgCache := getCacheHandle(cache.Config{Disable: disableCache})
	if imgCache == nil {
		sylog.Fatalf("Failed to create an image cache handle")
	}

	if syscall.Getuid() != 0 && !buildArgs.fakeroot && fs.IsFile(spec) && !isImage(spec) {
		sylog.Fatalf("You must be the root user, however you can use --remote or --fakeroot to build from a Singularity recipe file")
	}

	err := checkSections()
	if err != nil {
		sylog.Fatalf("Could not check build sections: %v", err)
	}

	authConf, err := makeDockerCredentials(cmd)
	if err != nil {
		sylog.Fatalf("While creating Docker credentials: %v", err)
	}

	// parse definition to determine build source
	defs, err := build.MakeAllDefs(spec)
	if err != nil {
		sylog.Fatalf("Unable to build from %s: %v", spec, err)
	}

	hasLibrary := false

	// only resolve remote endpoints if library is a build source
	for _, d := range defs {
		if d.Header["bootstrap"] == "library" {
			hasLibrary = true
			break
		}
	}

	authToken := ""

	if hasLibrary {
		lc, err := getLibraryClientConfig(buildArgs.libraryURL)
		if err != nil {
			sylog.Fatalf("Unable to get library client configuration: %v", err)
		}
		buildArgs.libraryURL = lc.BaseURL
		authToken = lc.AuthToken
	}

	co, err := getKeyserverClientOpts(buildArgs.keyServerURL, endpoint.KeyserverVerifyOp)
	if err != nil {
		sylog.Fatalf("Unable to get key server client configuration: %v", err)
	}

	buildFormat := "sif"
	sandboxTarget := false
	if buildArgs.sandbox {
		buildFormat = "sandbox"
		sandboxTarget = true

	}

	b, err := build.New(
		defs,
		build.Config{
			Dest:      dst,
			Format:    buildFormat,
			NoCleanUp: buildArgs.noCleanUp,
			Opts: types.Options{
				ImgCache:          imgCache,
				TmpDir:            tmpDir,
				NoCache:           disableCache,
				Update:            buildArgs.update,
				Force:             forceOverwrite,
				Sections:          buildArgs.sections,
				NoTest:            buildArgs.noTest,
				NoHTTPS:           noHTTPS,
				LibraryURL:        buildArgs.libraryURL,
				LibraryAuthToken:  authToken,
				KeyServerOpts:     co,
				DockerAuthConfig:  authConf,
				EncryptionKeyInfo: keyInfo,
				FixPerms:          buildArgs.fixPerms,
				SandboxTarget:     sandboxTarget,
			},
		})
	if err != nil {
		sylog.Fatalf("Unable to create build: %v", err)
	}

	if err = b.Full(ctx); err != nil {
		sylog.Fatalf("While performing build: %v", err)
	}
}

func checkSections() error {
	var all, none bool
	for _, section := range buildArgs.sections {
		if section == "none" {
			none = true
		}
		if section == "all" {
			all = true
		}
	}

	if all && len(buildArgs.sections) > 1 {
		return fmt.Errorf("section specification error: cannot have all and any other option")
	}
	if none && len(buildArgs.sections) > 1 {
		return fmt.Errorf("section specification error: cannot have none and any other option")
	}

	return nil
}

func isImage(spec string) bool {
	i, err := image.Init(spec, false)
	if i != nil {
		_ = i.File.Close()
	}
	return err == nil
}

// getEncryptionMaterial handles the setting of encryption environment and flag parameters to eventually be
// passed to the crypt package for handling.
// This handles the SINGULARITY_ENCRYPTION_PASSPHRASE/PEM_PATH envvars outside of cobra in order to
// enforce the unique flag/env precidence for the encryption flow
func getEncryptionMaterial(cmd *cobra.Command) (crypt.KeyInfo, error) {
	passphraseFlag := cmd.Flags().Lookup("passphrase")
	PEMFlag := cmd.Flags().Lookup("pem-path")
	passphraseEnv, passphraseEnvOK := os.LookupEnv("SINGULARITY_ENCRYPTION_PASSPHRASE")
	pemPathEnv, pemPathEnvOK := os.LookupEnv("SINGULARITY_ENCRYPTION_PEM_PATH")

	// checks for no flags/envvars being set
	if !(PEMFlag.Changed || pemPathEnvOK || passphraseFlag.Changed || passphraseEnvOK) {
		sylog.Fatalf("Unable to use container encryption. Must supply encryption material through environment variables or flags.")
	}

	// order of precidence:
	// 1. PEM flag
	// 2. Passphrase flag
	// 3. PEM envvar
	// 4. Passphrase envvar

	if PEMFlag.Changed {
		exists, err := fs.PathExists(encryptionPEMPath)
		if err != nil {
			sylog.Fatalf("Unable to verify existence of %s: %v", encryptionPEMPath, err)
		}

		if !exists {
			sylog.Fatalf("Specified PEM file %s: does not exist.", encryptionPEMPath)
		}

		sylog.Verbosef("Using pem path flag for encrypted container")

		// Check it's a valid PEM public key we can load, before starting the build (#4173)
		if cmd.Name() == "build" {
			if _, err := crypt.LoadPEMPublicKey(encryptionPEMPath); err != nil {
				sylog.Fatalf("Invalid encryption public key: %v", err)
			}
			// or a valid private key before launching the engine for actions on a container (#5221)
		} else {
			if _, err := crypt.LoadPEMPrivateKey(encryptionPEMPath); err != nil {
				sylog.Fatalf("Invalid encryption private key: %v", err)
			}
		}

		return crypt.KeyInfo{Format: crypt.PEM, Path: encryptionPEMPath}, nil
	}

	if passphraseFlag.Changed {
		sylog.Verbosef("Using interactive passphrase entry for encrypted container")
		passphrase, err := interactive.AskQuestionNoEcho("Enter encryption passphrase: ")
		if err != nil {
			return crypt.KeyInfo{}, err
		}
		if passphrase == "" {
			sylog.Fatalf("Cannot encrypt container with empty passphrase")
		}
		return crypt.KeyInfo{Format: crypt.Passphrase, Material: passphrase}, nil
	}

	if pemPathEnvOK {
		exists, err := fs.PathExists(pemPathEnv)
		if err != nil {
			sylog.Fatalf("Unable to verify existence of %s: %v", pemPathEnv, err)
		}

		if !exists {
			sylog.Fatalf("Specified PEM file %s: does not exist.", pemPathEnv)
		}

		sylog.Verbosef("Using pem path environment variable for encrypted container")
		return crypt.KeyInfo{Format: crypt.PEM, Path: pemPathEnv}, nil
	}

	if passphraseEnvOK {
		sylog.Verbosef("Using passphrase environment variable for encrypted container")
		return crypt.KeyInfo{Format: crypt.Passphrase, Material: passphraseEnv}, nil
	}

	return crypt.KeyInfo{}, nil
}

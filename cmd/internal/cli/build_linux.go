// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
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
	"syscall"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/build/remotebuilder"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config"
	fakerootConfig "github.com/sylabs/singularity/internal/pkg/runtime/engine/fakeroot/config"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/interactive"
	"github.com/sylabs/singularity/internal/pkg/util/starter"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/image"
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
	buildFormat := "sif"
	if sandbox {
		buildFormat = "sandbox"
	}

	if buildArch != runtime.GOARCH && !remote {
		sylog.Fatalf("Requested architecture (%s) does not match host (%s). Cannot build locally.", buildArch, runtime.GOARCH)
	}

	dest := args[0]
	spec := args[1]

	// check if target collides with existing file
	if err := checkBuildTarget(dest); err != nil {
		sylog.Fatalf("%s", err)
	}

	if remote {
		// building encrypted containers on the remote builder is not currently supported
		if encrypt {
			sylog.Fatalf("Building encrypted container with the remote builder is not currently supported.")
		}

		handleRemoteBuildFlags(cmd)

		// submitting a remote build requires a valid authToken
		if authToken == "" {
			sylog.Fatalf("Unable to submit build job: %v", remoteWarning)
		}

		def, err := definitionFromSpec(spec)
		if err != nil {
			sylog.Fatalf("Unable to build from %s: %v", spec, err)
		}

		if sandbox {
			// create temporary file to download sif
			f, err := ioutil.TempFile(tmpDir, "remote-build-")
			if err != nil {
				sylog.Fatalf("Could not create temporary directory: %s", err)
			}
			os.Remove(f.Name())
			dest = f.Name()

			// remove downloaded sif
			defer os.Remove(f.Name())

			// build from sif downloaded in tmp location
			defer func() {
				sylog.Debugf("Building sandbox from downloaded SIF")
				imgCache := getCacheHandle(cache.Config{})
				if imgCache == nil {
					sylog.Fatalf("failed to create an image cache handle")
				}

				d, err := types.NewDefinitionFromURI("localimage" + "://" + dest)
				if err != nil {
					sylog.Fatalf("Unable to create definition for sandbox build: %v", err)
				}

				b, err := build.New(
					[]types.Definition{d},
					build.Config{
						Dest:      args[0],
						Format:    buildFormat,
						NoCleanUp: noCleanUp,
						Opts: types.Options{
							ImgCache: imgCache,
							NoCache:  disableCache,
							TmpDir:   tmpDir,
							Update:   update,
							Force:    force,
						},
					})
				if err != nil {
					sylog.Fatalf("Unable to create build: %v", err)
				}

				if err = b.Full(); err != nil {
					sylog.Fatalf("While performing build: %v", err)
				}
			}()
		}

		b, err := remotebuilder.New(dest, libraryURL, def, detached, force, builderURL, authToken, buildArch)
		if err != nil {
			sylog.Fatalf("Failed to create builder: %v", err)
		}
		err = b.Build(context.TODO())
		if err != nil {
			sylog.Fatalf("While performing build: %v", err)
		}
	} else {
		var keyInfo *crypt.KeyInfo
		if encrypt || enterPassphrase || cmd.Flags().Lookup("pem-path").Changed {
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

		imgCache := getCacheHandle(cache.Config{})
		if imgCache == nil {
			sylog.Fatalf("Failed to create an image cache handle")
		}

		if syscall.Getuid() != 0 && !fakeroot && fs.IsFile(spec) && !isImage(spec) {
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

		// only resolve remote endpoints if library is a build source
		for _, d := range defs {
			if d.Header["bootstrap"] == "library" {
				handleBuildFlags(cmd)
				continue
			}
		}

		b, err := build.New(
			defs,
			build.Config{
				Dest:      dest,
				Format:    buildFormat,
				NoCleanUp: noCleanUp,
				Opts: types.Options{
					ImgCache:          imgCache,
					TmpDir:            tmpDir,
					NoCache:           disableCache,
					Update:            update,
					Force:             force,
					Sections:          sections,
					NoTest:            noTest,
					NoHTTPS:           noHTTPS,
					LibraryURL:        libraryURL,
					LibraryAuthToken:  authToken,
					DockerAuthConfig:  authConf,
					EncryptionKeyInfo: keyInfo,
				},
			})
		if err != nil {
			sylog.Fatalf("Unable to create build: %v", err)
		}

		if err = b.Full(); err != nil {
			sylog.Fatalf("While performing build: %v", err)
		}
	}
	sylog.Infof("Build complete: %s", dest)
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
		return fmt.Errorf("section specification error: cannot have all and any other option")
	}
	if none && len(sections) > 1 {
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

// standard builds should just warn and fall back to CLI default if we cannot resolve library URL
func handleBuildFlags(cmd *cobra.Command) {
	// if we can load config and if default endpoint is set, use that
	// otherwise fall back on regular authtoken and URI behavior
	endpoint, err := sylabsRemote(remoteConfig)
	if err == scs.ErrNoDefault {
		sylog.Warningf("No default remote in use, falling back to %v", libraryURL)
		return
	} else if err != nil {
		sylog.Fatalf("Unable to load remote configuration: %v", err)
	}

	authToken = endpoint.Token
	if !cmd.Flags().Lookup("library").Changed {
		uri, err := endpoint.GetServiceURI("library")
		if err == nil {
			libraryURL = uri
		} else {
			sylog.Warningf("Unable to get library service URI: %v", err)
		}
	}
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
		sylog.Fatalf("Unable to use container encryption. Must supply encryption material through enironment variables or flags.")
	}

	// order of precidence:
	// 1. PEM flag
	// 2. Passphrase flag
	// 3. PEM envvar
	// 4. Passphrase envvar

	if PEMFlag.Changed {
		exists, err := fs.FileExists(encryptionPEMPath)
		if err != nil {
			sylog.Fatalf("Unable to verify existence of %s: %v", encryptionPEMPath, err)
		}

		if !exists {
			sylog.Fatalf("Specified PEM file %s: does not exist.", encryptionPEMPath)
		}

		sylog.Verbosef("Using pem path flag for encrypted container")
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
		exists, err := fs.FileExists(pemPathEnv)
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

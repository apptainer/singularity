// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	osExec "os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/build/remotebuilder"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	fakerootConfig "github.com/sylabs/singularity/internal/pkg/runtime/engines/fakeroot/config"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/exec"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/image"
)

func fakerootExec(cmdArgs []string) {
	starter := filepath.Join(buildcfg.LIBEXECDIR, "singularity/bin/starter-suid")

	// singularity was compiled with '--without-suid' option
	if buildcfg.SINGULARITY_SUID_INSTALL == 0 {
		starter = filepath.Join(buildcfg.LIBEXECDIR, "singularity/bin/starter")
	}
	if _, err := os.Stat(starter); os.IsNotExist(err) {
		sylog.Fatalf("%s not found, please check your installation", starter)
	}

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
		Args: args,
		Envs: os.Environ(),
		Home: user.Dir,
	}

	cfg := &config.Common{
		EngineName:   fakerootConfig.Name,
		ContainerID:  "fakeroot",
		EngineConfig: engineConfig,
	}

	configData, err := json.Marshal(cfg)
	if err != nil {
		sylog.Fatalf("CLI Failed to marshal CommonEngineConfig: %s\n", err)
	}

	Env := []string{"SINGULARITY_MESSAGELEVEL=0"}

	if err := exec.Pipe(starter, []string{"Singularity fakeroot"}, Env, configData); err != nil {
		sylog.Fatalf("%s", err)
	}
}

func run(cmd *cobra.Command, args []string) {
	buildFormat := "sif"
	if sandbox {
		buildFormat = "sandbox"
	}

	if buildArch != runtime.GOARCH && !remote {
		sylog.Fatalf("Requested architecture (%s) does not match host (%s). Cannot build locally.", buildArch, runtime.GOARCH)
		cmd.Flags().Lookup("arch").Value.Set(runtime.GOARCH)
	}

	dest := args[0]
	spec := args[1]

	// check if target collides with existing file
	if ok := checkBuildTarget(dest, update); !ok {
		os.Exit(1)
	}

	if remote {
		handleRemoteBuildFlags(cmd)

		// Submiting a remote build requires a valid authToken
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
		imgCache := getCacheHandle(cache.Config{})
		if imgCache == nil {
			sylog.Fatalf("failed to create an image cache handle")
		}

		if syscall.Getuid() != 0 && !fakeroot && fs.IsFile(spec) && !isImage(spec) {
			sylog.Fatalf("You must be the root user, however you can use --remote or --fakeroot to build from a Singularity recipe file")
		}

		err := checkSections()
		if err != nil {
			sylog.Fatalf(err.Error())
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
			if d.Header != nil && d.Header["bootstrap"] == "library" {
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
					ImgCache:         imgCache,
					TmpDir:           tmpDir,
					NoCache:          disableCache,
					Update:           update,
					Force:            force,
					Sections:         sections,
					NoTest:           noTest,
					NoHTTPS:          noHTTPS,
					LibraryURL:       libraryURL,
					LibraryAuthToken: authToken,
					DockerAuthConfig: authConf,
					EncryptionKey:    encryptionKey,
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
	_, err := image.Init(spec, false)
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

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
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/util/fs"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/build/remotebuilder"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
)

func run(cmd *cobra.Command, args []string) {
	buildFormat := "sif"
	if sandbox {
		buildFormat = "sandbox"
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
				imgCache := getCacheHandle()
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
						ImgCache:  imgCache,
						Dest:      args[0],
						Format:    buildFormat,
						NoCleanUp: noCleanUp,
						Opts: types.Options{
							TmpDir: tmpDir,
							Update: update,
							Force:  force,
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

		b, err := remotebuilder.New(dest, libraryURL, def, detached, force, builderURL, authToken)
		if err != nil {
			sylog.Fatalf("Failed to create builder: %v", err)
		}
		err = b.Build(context.TODO())
		if err != nil {
			sylog.Fatalf("While performing build: %v", err)
		}
	} else {
		if syscall.Getuid() != 0 && !fakeroot && fs.IsFile(spec) {
			sylog.Fatalf("You must be the root user, however you can use --remote or --fakeroot to build from a Singularity recipe file")
		}


		imgCache := getCacheHandle()
		if imgCache == nil {
			sylog.Fatalf("failed to create an image cache handle")
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
				ImgCache:  imgCache,
				Dest:      dest,
				Format:    buildFormat,
				NoCleanUp: noCleanUp,
				Opts: types.Options{
					TmpDir:           tmpDir,
					Update:           update,
					Force:            force,
					Sections:         sections,
					NoTest:           noTest,
					NoHTTPS:          noHTTPS,
					LibraryURL:       libraryURL,
					LibraryAuthToken: authToken,
					DockerAuthConfig: authConf,
					Fakeroot:         fakeroot,
				},
			})
		if err != nil {
			sylog.Fatalf("Unable to create build: %v", err)
		}

		if err = b.Full(); err != nil {
			sylog.Fatalf("While performing build: %v", err)
		}
	}
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

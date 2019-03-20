// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/build/remotebuilder"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
)

func preRun(cmd *cobra.Command, args []string) {
	sylabsToken(cmd, args)
}

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
			sylog.Fatalf("Unable to submit build job: %v", authWarning)
		}

		def, err := definitionFromSpec(spec)
		if err != nil {
			sylog.Fatalf("Unable to build from %s: %v", spec, err)
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
		err := checkSections()
		if err != nil {
			sylog.Fatalf(err.Error())
		}

		authConf, err := makeDockerCredentials(cmd)
		if err != nil {
			sylog.Fatalf("While creating Docker credentials: %v", err)
		}

		// parse definition to determine build source
		def, err := build.MakeDef(spec, false)
		if err != nil {
			sylog.Fatalf("Unable to build from %s: %v", spec, err)
		}

		// only resolve remote endpoints if library is the build source
		if def.Header["bootstrap"] == "library" {
			handleBuildFlags(cmd)
		}

		b, err := build.NewBuild(
			spec,
			dest,
			buildFormat,
			libraryURL,
			authToken,
			types.Options{
				TmpDir:           tmpDir,
				Update:           update,
				Force:            force,
				Sections:         sections,
				NoTest:           noTest,
				NoHTTPS:          noHTTPS,
				NoCleanUp:        noCleanUp,
				DockerAuthConfig: authConf,
			},
			AllowUnauthenticatedBuild)
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
		} else if err != nil {
			sylog.Warningf("Unable to get library service URI: %v", err)
		}
	}
}

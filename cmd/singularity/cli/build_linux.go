// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/build/remotebuilder"
	"github.com/sylabs/singularity/internal/pkg/build/types"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/syplugin"
)

func preRun(cmd *cobra.Command, args []string) {
	sylabsToken(cmd, args)
	syplugin.Init()
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

		b, err := build.NewBuild(
			spec,
			dest,
			buildFormat,
			libraryURL,
			authToken,
			types.Options{
				TmpDir:   tmpDir,
				Update:   update,
				Force:    force,
				Sections: sections,
				NoTest:   noTest,
				NoHTTPS:  noHTTPS,
			})
		if err != nil {
			sylog.Fatalf("Unable to create build: %v", err)
		}

		if err = b.Full(); err != nil {
			sylog.Fatalf("While performing build: %v", err)
		}
	}
}

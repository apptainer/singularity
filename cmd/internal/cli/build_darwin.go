// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/build/remotebuilder"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func fakerootExec(cmdArgs []string) {
	sylog.Fatalf("fakeroot is not supported on this platform")
}

func runBuild(cmd *cobra.Command, args []string) {
	dest := args[0]
	spec := args[1]

	// check if target collides with existing file
	if err := checkBuildTarget(dest); err != nil {
		sylog.Fatalf("%s", err)
	}

	if !remote {
		sylog.Fatalf("Only remote builds are supported on this platform")
	}

	handleRemoteBuildFlags(cmd)

	// Submiting a remote build requires a valid authToken
	if authToken == "" {
		sylog.Fatalf("Unable to submit build job: %v", remoteWarning)
	}

	def, err := definitionFromSpec(spec)
	if err != nil {
		sylog.Fatalf("Unable to build from %s: %v", spec, err)
	}

	b, err := remotebuilder.New(dest, libraryURL, def, detached, force, builderURL, authToken, buildArch)
	if err != nil {
		sylog.Fatalf("Failed to create builder: %v", err)
	}

	err = b.Build(context.TODO())
	if err != nil {
		sylog.Fatalf("While performing build: %v", err)
	}
}

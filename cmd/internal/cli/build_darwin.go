// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/build/remotebuilder"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func preRun(cmd *cobra.Command, args []string) {
	sylabsToken(cmd, args)
}

func run(cmd *cobra.Command, args []string) {
	dest := args[0]
	spec := args[1]

	// check if target collides with existing file
	if ok := checkBuildTarget(dest, false); !ok {
		os.Exit(1)
	}

	if !remote && !cmd.Flags().Lookup("builder").Changed {
		sylog.Fatalf("Only remote builds are supported on this platform")
	}

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
}

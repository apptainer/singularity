// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package pull

import (
	"path"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
)

// If a remote is set to a different endpoint we should be able to pull
// with `--library https://library.sylabs.io` from the default Sylabs cloud library.
func (c ctx) issue5808(t *testing.T) {
	testEndpoint := "issue5808"
	testEndpointURI := "https://cloud.staging.sylabs.io"
	defaultLibraryURI := "https://library.sylabs.io"
	testImage := "library://sylabs/tests/signed:1.0.0"

	pullDir, cleanup := e2e.MakeTempDir(t, "", "issue5808", "")
	defer cleanup(t)

	// Add another endpoint
	argv := []string{"add", "--no-login", testEndpoint, testEndpointURI}
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("remote add"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("remote"),
		e2e.WithArgs(argv...),
		e2e.ExpectExit(0),
	)
	// Remove test remote when we are done here
	defer func(t *testing.T) {
		argv := []string{"remove", testEndpoint}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest("remote remove"),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(0),
		)
	}(t)

	// Set as default
	argv = []string{"use", testEndpoint}
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("remote use"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("remote"),
		e2e.WithArgs(argv...),
		e2e.ExpectExit(0),
	)

	// Pull a library image
	dest := path.Join(pullDir, "alpine.sif")
	argv = []string{"--arch", "amd64", "--library", defaultLibraryURI, dest, testImage}
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("pull"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("pull"),
		e2e.WithArgs(argv...),
		e2e.ExpectExit(0),
	)

}

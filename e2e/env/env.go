// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// This test sets singularity image specific environment variables and
// verifies that they are properly set.

package singularityenv

import (
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
)

type ctx struct {
	env e2e.TestEnv
}

func (c ctx) singularityEnv(t *testing.T) {
	// Singularity defines a path by default. See singularityware/singularity/etc/init.
	var defaultImage = "docker://alpine:3.8"
	var defaultPath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

	// This image sets a custom path.
	var customImage = "docker://godlovedc/lolcow"
	var customPath = "/usr/games:" + defaultPath

	// Append or prepend this path.
	var partialPath = "/foo"

	// Overwrite the path with this one.
	var overwrittenPath = "/usr/bin:/bin"

	var tests = []struct {
		name  string
		image string
		path  string
		env   []string
	}{
		{
			name:  "DefaultPath",
			image: defaultImage,
			path:  defaultPath,
			env:   []string{},
		},
		{
			name:  "CustomPath",
			image: customImage,
			path:  customPath,
			env:   []string{},
		},
		{
			name:  "AppendToDefaultPath",
			image: defaultImage,
			path:  defaultPath + ":" + partialPath,
			env:   []string{"SINGULARITYENV_APPEND_PATH=/foo"},
		},
		{
			name:  "AppendToCustomPath",
			image: customImage,
			path:  customPath + ":" + partialPath,
			env:   []string{"SINGULARITYENV_APPEND_PATH=/foo"},
		},
		{
			name:  "PrependToDefaultPath",
			image: defaultImage,
			path:  partialPath + ":" + defaultPath,
			env:   []string{"SINGULARITYENV_PREPEND_PATH=/foo"},
		},
		{
			name:  "PrependToCustomPath",
			image: customImage,
			path:  partialPath + ":" + customPath,
			env:   []string{"SINGULARITYENV_PREPEND_PATH=/foo"},
		},
		{
			name:  "OverwriteDefaultPath",
			image: defaultImage,
			path:  overwrittenPath,
			env:   []string{"SINGULARITYENV_PATH=" + overwrittenPath},
		},
		{
			name:  "OverwriteCustomPath",
			image: customImage,
			path:  overwrittenPath,
			env:   []string{"SINGULARITYENV_PATH=" + overwrittenPath},
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("exec"),
			e2e.WithEnv(tt.env),
			e2e.WithArgs(tt.image, "env"),
			e2e.ExpectExit(
				0,
				e2e.ExpectOutput(e2e.ContainMatch, tt.path),
			),
		)
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := ctx{
		env: env,
	}

	return testhelper.TestRunner(map[string]func(*testing.T){
		"environment manipulation": c.singularityEnv,
	})
}

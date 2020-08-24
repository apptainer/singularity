// Copyright (c) 2019-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// This test sets singularity image specific environment variables and
// verifies that they are properly set.

package singularityenv

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
)

type ctx struct {
	env e2e.TestEnv
}

const (
	defaultPath     = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	singularityLibs = "/.singularity.d/libs"
)

func (c ctx) singularityEnv(t *testing.T) {
	// use a cache to not download images over and over
	imgCacheDir, cleanCache := e2e.MakeCacheDir(t, c.env.TestDir)
	defer cleanCache(t)
	c.env.ImgCacheDir = imgCacheDir

	// Singularity defines a path by default. See singularityware/singularity/etc/init.
	var defaultImage = "docker://alpine:3.8"

	// This image sets a custom path.
	var customImage = "docker://sylabsio/lolcow"
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
			e2e.WithArgs(tt.image, "/bin/sh", "-c", "echo $PATH"),
			e2e.ExpectExit(
				0,
				e2e.ExpectOutput(e2e.ExactMatch, tt.path),
			),
		)
	}
}

func (c ctx) singularityEnvOption(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	imageDefaultPath := defaultPath + ":/go/bin:/usr/local/go/bin"

	// use a cache to not download images over and over
	imgCacheDir, cleanCache := e2e.MakeCacheDir(t, c.env.TestDir)
	defer cleanCache(t)
	c.env.ImgCacheDir = imgCacheDir

	var tests = []struct {
		name     string
		image    string
		envOpt   []string
		hostEnv  []string
		matchEnv string
		matchVal string
	}{
		{
			name:     "DefaultPath",
			image:    "docker://alpine:3.8",
			matchEnv: "PATH",
			matchVal: defaultPath,
		},
		{
			name:     "DefaultPathOverride",
			image:    "docker://alpine:3.8",
			envOpt:   []string{"PATH=/"},
			matchEnv: "PATH",
			matchVal: "/",
		},
		{
			name:     "AppendDefaultPath",
			image:    "docker://alpine:3.8",
			envOpt:   []string{"APPEND_PATH=/foo"},
			matchEnv: "PATH",
			matchVal: defaultPath + ":/foo",
		},
		{
			name:     "PrependDefaultPath",
			image:    "docker://alpine:3.8",
			envOpt:   []string{"PREPEND_PATH=/foo"},
			matchEnv: "PATH",
			matchVal: "/foo:" + defaultPath,
		},
		{
			name:     "DockerImage",
			image:    "docker://sylabsio/lolcow",
			matchEnv: "LC_ALL",
			matchVal: "C",
		},
		{
			name:     "DockerImageOverride",
			image:    "docker://sylabsio/lolcow",
			envOpt:   []string{"LC_ALL=foo"},
			matchEnv: "LC_ALL",
			matchVal: "foo",
		},
		{
			name:     "DefaultPathTestImage",
			image:    c.env.ImagePath,
			matchEnv: "PATH",
			matchVal: imageDefaultPath,
		},
		{
			name:     "DefaultPathTestImageOverride",
			image:    c.env.ImagePath,
			envOpt:   []string{"PATH=/"},
			matchEnv: "PATH",
			matchVal: "/",
		},
		{
			name:     "AppendDefaultPathTestImage",
			image:    c.env.ImagePath,
			envOpt:   []string{"APPEND_PATH=/foo"},
			matchEnv: "PATH",
			matchVal: imageDefaultPath + ":/foo",
		},
		{
			name:     "AppendLiteralDefaultPathTestImage",
			image:    c.env.ImagePath,
			envOpt:   []string{"PATH=$PATH:/foo"},
			matchEnv: "PATH",
			matchVal: imageDefaultPath + ":/foo",
		},
		{
			name:     "PrependDefaultPathTestImage",
			image:    c.env.ImagePath,
			envOpt:   []string{"PREPEND_PATH=/foo"},
			matchEnv: "PATH",
			matchVal: "/foo:" + imageDefaultPath,
		},
		{
			name:     "PrependLiteralDefaultPathTestImage",
			image:    c.env.ImagePath,
			envOpt:   []string{"PATH=/foo:$PATH"},
			matchEnv: "PATH",
			matchVal: "/foo:" + imageDefaultPath,
		},
		{
			name:     "TestImageCgoEnabledDefault",
			image:    c.env.ImagePath,
			matchEnv: "CGO_ENABLED",
			matchVal: "0",
		},
		{
			name:     "TestImageCgoEnabledOverride",
			image:    c.env.ImagePath,
			envOpt:   []string{"CGO_ENABLED=1"},
			matchEnv: "CGO_ENABLED",
			matchVal: "1",
		},
		{
			name:     "TestImageCgoEnabledOverride_KO",
			image:    c.env.ImagePath,
			hostEnv:  []string{"CGO_ENABLED=1"},
			matchEnv: "CGO_ENABLED",
			matchVal: "0",
		},
		{
			name:     "TestImageCgoEnabledOverrideFromEnv",
			image:    c.env.ImagePath,
			hostEnv:  []string{"SINGULARITYENV_CGO_ENABLED=1"},
			matchEnv: "CGO_ENABLED",
			matchVal: "1",
		},
		{
			name:     "TestImageCgoEnabledOverrideEnvOptionPrecedence",
			image:    c.env.ImagePath,
			hostEnv:  []string{"SINGULARITYENV_CGO_ENABLED=1"},
			envOpt:   []string{"CGO_ENABLED=2"},
			matchEnv: "CGO_ENABLED",
			matchVal: "2",
		},
		{
			name:     "TestImageCgoEnabledOverrideEmpty",
			image:    c.env.ImagePath,
			envOpt:   []string{"CGO_ENABLED="},
			matchEnv: "CGO_ENABLED",
			matchVal: "",
		},
		{
			name:     "TestImageOverrideHost",
			image:    c.env.ImagePath,
			hostEnv:  []string{"FOO=bar"},
			envOpt:   []string{"FOO=foo"},
			matchEnv: "FOO",
			matchVal: "foo",
		},
		{
			name:     "TestMultiLine",
			image:    c.env.ImagePath,
			hostEnv:  []string{"MULTI=Hello\nWorld"},
			matchEnv: "MULTI",
			matchVal: "Hello\nWorld",
		},
		{
			name:  "TestInvalidKey",
			image: c.env.ImagePath,
			// We try to set an invalid env var... and make sure
			// we have no error output from the interpreter as it
			// should be ignored, not passed into the container.
			hostEnv:  []string{"BASH_FUNC_ml%%=TEST"},
			matchEnv: "BASH_FUNC_ml%%",
			matchVal: "",
		},
		{
			name:     "TestDefaultLdLibraryPath",
			image:    c.env.ImagePath,
			matchEnv: "LD_LIBRARY_PATH",
			matchVal: singularityLibs,
		},
		{
			name:     "TestCustomLdLibraryPath",
			image:    c.env.ImagePath,
			envOpt:   []string{"LD_LIBRARY_PATH=/foo"},
			matchEnv: "LD_LIBRARY_PATH",
			matchVal: "/foo:" + singularityLibs,
		},
	}

	for _, tt := range tests {
		args := make([]string, 0)
		if tt.envOpt != nil {
			args = append(args, "--env", strings.Join(tt.envOpt, ","))
		}
		args = append(args, tt.image, "/bin/sh", "-c", "echo \"${"+tt.matchEnv+"}\"")
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("exec"),
			e2e.WithEnv(tt.hostEnv),
			e2e.WithArgs(args...),
			e2e.ExpectExit(
				0,
				e2e.ExpectOutput(e2e.ExactMatch, tt.matchVal),
			),
		)
	}
}

func (c ctx) singularityEnvFile(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	imageDefaultPath := defaultPath + ":/go/bin:/usr/local/go/bin"

	dir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "envfile-", "")
	defer cleanup(t)
	p := filepath.Join(dir, "env.file")

	// use a cache to not download images over and over
	imgCacheDir, cleanCache := e2e.MakeCacheDir(t, c.env.TestDir)
	defer cleanCache(t)
	c.env.ImgCacheDir = imgCacheDir

	var tests = []struct {
		name     string
		image    string
		envFile  string
		envOpt   []string
		hostEnv  []string
		matchEnv string
		matchVal string
	}{
		{
			name:     "DefaultPathOverride",
			image:    c.env.ImagePath,
			envFile:  "PATH=/",
			matchEnv: "PATH",
			matchVal: "/",
		},
		{
			name:     "DefaultPathOverrideEnvOptionPrecedence",
			image:    c.env.ImagePath,
			envOpt:   []string{"PATH=/etc"},
			envFile:  "PATH=/",
			matchEnv: "PATH",
			matchVal: "/etc",
		},
		{
			name:     "DefaultPathOverrideEnvOptionPrecedence",
			image:    c.env.ImagePath,
			envOpt:   []string{"PATH=/etc"},
			envFile:  "PATH=/",
			matchEnv: "PATH",
			matchVal: "/etc",
		},
		{
			name:     "AppendDefaultPath",
			image:    c.env.ImagePath,
			envFile:  "APPEND_PATH=/",
			matchEnv: "PATH",
			matchVal: imageDefaultPath + ":/",
		},
		{
			name:     "AppendLiteralDefaultPath",
			image:    c.env.ImagePath,
			envFile:  `PATH="\$PATH:/"`,
			matchEnv: "PATH",
			matchVal: imageDefaultPath + ":/",
		},
		{
			name:     "PrependLiteralDefaultPath",
			image:    c.env.ImagePath,
			envFile:  `PATH="/:\$PATH"`,
			matchEnv: "PATH",
			matchVal: "/:" + imageDefaultPath,
		},
		{
			name:     "PrependDefaultPath",
			image:    c.env.ImagePath,
			envFile:  "PREPEND_PATH=/",
			matchEnv: "PATH",
			matchVal: "/:" + imageDefaultPath,
		},
		{
			name:     "DefaultLdLibraryPath",
			image:    c.env.ImagePath,
			matchEnv: "LD_LIBRARY_PATH",
			matchVal: singularityLibs,
		},
		{
			name:     "CustomLdLibraryPath",
			image:    c.env.ImagePath,
			envFile:  "LD_LIBRARY_PATH=/foo",
			matchEnv: "LD_LIBRARY_PATH",
			matchVal: "/foo:" + singularityLibs,
		},
	}

	for _, tt := range tests {
		args := make([]string, 0)
		if tt.envOpt != nil {
			args = append(args, "--env", strings.Join(tt.envOpt, ","))
		}
		if tt.envFile != "" {
			ioutil.WriteFile(p, []byte(tt.envFile), 0644)
			args = append(args, "--env-file", p)
		}
		args = append(args, tt.image, "/bin/sh", "-c", "echo $"+tt.matchEnv)

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("exec"),
			e2e.WithEnv(tt.hostEnv),
			e2e.WithArgs(args...),
			e2e.ExpectExit(
				0,
				e2e.ExpectOutput(e2e.ExactMatch, tt.matchVal),
			),
		)
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) testhelper.Tests {
	c := ctx{
		env: env,
	}

	return testhelper.Tests{
		"environment manipulation": c.singularityEnv,
		"environment option":       c.singularityEnvOption,
		"environment file":         c.singularityEnvFile,
		"issue 5057":               c.issue5057, // https://github.com/sylabs/hpcng/issues/5057
		"issue 5426":               c.issue5426, // https://github.com/sylabs/hpcng/issues/5426
	}
}

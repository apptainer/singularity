// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build e2e_scripts

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/pkg/stest"

	// custom builtins
	_ "github.com/sylabs/singularity/e2e/scripts_builtins/command"
	_ "github.com/sylabs/singularity/e2e/scripts_builtins/test"
)

var testScripts = []struct {
	name        string
	path        string
	runParallel bool
}{
	// Build tests
	{
		name:        "BUILD/BASIC",
		path:        "build/basic.test",
		runParallel: true,
	},
	// Actions tests
	{
		name:        "ACTIONS/ENV_PATH",
		path:        "actions/env_path.test",
		runParallel: true,
	},
	{
		name:        "ACTIONS/RUN",
		path:        "actions/run.test",
		runParallel: true,
	},
	{
		name:        "ACTIONS/EXEC",
		path:        "actions/exec.test",
		runParallel: true,
	},
	{
		name:        "ACTIONS/FROM_URI",
		path:        "actions/from_uri.test",
		runParallel: true,
	},
	{
		name:        "ACTIONS/STDOUT",
		path:        "actions/stdout.test",
		runParallel: true,
	},
	{
		name:        "ACTIONS/STDIN",
		path:        "actions/stdin.test",
		runParallel: true,
	},
	{
		name:        "ACTIONS/PERSISTENT_OVERLAY",
		path:        "actions/persistent_overlay.test",
		runParallel: true,
	},
	// Docker tests
	{
		name:        "DOCKER/PULL",
		path:        "docker/pull.test",
		runParallel: true,
	},
	{
		name:        "DOCKER/DEFINITION",
		path:        "docker/definition.test",
		runParallel: true,
	},
	{
		name:        "DOCKER/AUFS",
		path:        "docker/aufs.test",
		runParallel: true,
	},
	{
		name:        "DOCKER/REGISTRY",
		path:        "docker/registry.test",
		runParallel: true,
	},
	// Instance tests
	{
		name:        "INSTANCE/BASIC_ECHO",
		path:        "instance/basic_echo.test",
		runParallel: true,
	},
	{
		name:        "INSTANCE/BASIC_OPTIONS",
		path:        "instance/basic_options.test",
		runParallel: true,
	},
	{
		name:        "INSTANCE/FROM_URI",
		path:        "instance/from_uri.test",
		runParallel: true,
	},
	{
		name:        "INSTANCE/CONTAIN",
		path:        "instance/contain.test",
		runParallel: true,
	},
	{
		name:        "INSTANCE/CREATE_MANY",
		path:        "instance/create_many.test",
		runParallel: true,
	},
	// OCI tests
	{
		name:        "OCI/BASIC",
		path:        "oci/basic.test",
		runParallel: true,
	},
	{
		name:        "OCI/ATTACH",
		path:        "oci/attach.test",
		runParallel: true,
	},
	{
		name:        "OCI/RUN",
		path:        "oci/run.test",
		runParallel: true,
	},
	{
		name:        "EXAMPLE",
		path:        "example/example.test",
		runParallel: true,
	},
}

func sudoExec(sudo string, args []string) error {
	cmd := exec.Command(sudo, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo %s failed: %s", strings.Join(args, " "), err)
	}
	return nil
}

func TestE2E(t *testing.T) {
	const (
		testDirPrefix   = "stest-"
		privDirPrefix   = "priv-"
		unprivDirPrefix = "unpriv-"
		defaultDirMode  = 0755
	)

	sudo, err := exec.LookPath("sudo")
	if err != nil {
		t.Fatalf("sudo executable not found in $PATH")
	}

	// first sudo run to ask for password if required
	if err := sudoExec(sudo, []string{"true"}); err != nil {
		t.Fatalf("%s", err)
	}

	// maintain sudo session for use in test scripts without
	// password ask
	go func() {
		time.Sleep(1 * time.Minute)
		if err := sudoExec(sudo, []string{"true"}); err != nil {
			t.Fatalf("%s", err)
		}
	}()

	testDir, err := e2e.MakeTmpDir("", testDirPrefix, defaultDirMode)
	if err != nil {
		t.Fatalf("%s", err)
	}

	// use sudo here to remove test directory in order to
	// delete image/files/directories that could be created
	// by privileged run/tests
	defer sudoExec(sudo, []string{"rm", "-rf", testDir})

	sourceDir, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("failed to determine absolute source directory: %s", err)
	}

	if err := os.Chdir("scripts"); err != nil {
		t.Fatalf("could not chdir to 'scripts' directory: %s", err)
	}

	os.Setenv("SUDO", sudo)
	os.Setenv("TESTDIR", testDir)
	os.Setenv("SOURCEDIR", sourceDir)
	os.Setenv("GOOS", runtime.GOOS)
	os.Setenv("GOARCH", runtime.GOARCH)

	if testing.Verbose() {
		os.Setenv("VERBOSE_TEST", "1")

		fmt.Println("Available environment variable in test script:")
		fmt.Printf("SUDO: %s\n", sudo)
		fmt.Printf("TESTDIR: %s\n", testDir)
		fmt.Printf("SOURCEDIR: %s\n", sourceDir)
		fmt.Printf("PATH: %s\n", os.Getenv("PATH"))
	} else {
		os.Setenv("VERBOSE_TEST", "0")
	}

	syscall.Umask(0022)

	parallel := false
	for _, a := range os.Args {
		if strings.Contains(a, "test.parallel=") {
			parallel = true
			break
		}
	}

	t.Run("SCRIPTS", func(t *testing.T) {
		for _, ts := range testScripts {
			if ts.runParallel {
				path := ts.path
				t.Run(ts.name, func(t *testing.T) {
					if parallel {
						t.Parallel()
					}
					stest.RunScript(nil, path, t)
				})
			}
		}
	})

	t.Run("SCRIPTS", func(t *testing.T) {
		for _, ts := range testScripts {
			if !ts.runParallel {
				path := ts.path
				t.Run(ts.name, func(t *testing.T) {
					stest.RunScript(nil, path, t)
				})
			}
		}
	})
}

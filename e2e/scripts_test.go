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
	"testing"
	"time"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/pkg/stest"

	// custom builtins
	_ "github.com/sylabs/singularity/e2e/scripts_builtins/command"
	_ "github.com/sylabs/singularity/e2e/scripts_builtins/test"
)

var testScripts = []struct {
	name string
	path string
}{
	// Build tests
	{
		name: "BUILD/BASIC",
		path: "build/basic.test",
	},
	// Actions tests
	{
		name: "ACTIONS/ENV_PATH",
		path: "actions/env_path.test",
	},
	{
		name: "ACTIONS/RUN",
		path: "actions/run.test",
	},
	{
		name: "ACTIONS/EXEC",
		path: "actions/exec.test",
	},
	{
		name: "ACTIONS/FROM_URI",
		path: "actions/from_uri.test",
	},
	{
		name: "ACTIONS/STDOUT",
		path: "actions/stdout.test",
	},
	{
		name: "ACTIONS/STDIN",
		path: "actions/stdin.test",
	},
	{
		name: "ACTIONS/PERSISTENT_OVERLAY",
		path: "actions/persistent_overlay.test",
	},
	// Docker tests
	{
		name: "DOCKER/PULL",
		path: "docker/pull.test",
	},
	{
		name: "DOCKER/DEFINITION",
		path: "docker/definition.test",
	},
	{
		name: "DOCKER/AUFS",
		path: "docker/aufs.test",
	},
	{
		name: "DOCKER/REGISTRY",
		path: "docker/registry.test",
	},
	// Instance tests
	{
		name: "INSTANCE/BASIC_ECHO",
		path: "instance/basic_echo.test",
	},
	{
		name: "INSTANCE/BASIC_OPTIONS",
		path: "instance/basic_options.test",
	},
	{
		name: "INSTANCE/FROM_URI",
		path: "instance/from_uri.test",
	},
	{
		name: "INSTANCE/CONTAIN",
		path: "instance/contain.test",
	},
	{
		name: "INSTANCE/CREATE_MANY",
		path: "instance/create_many.test",
	},
	// OCI tests
	{
		name: "OCI/BASIC",
		path: "oci/basic.test",
	},
	{
		name: "OCI/ATTACH",
		path: "oci/attach.test",
	},
	{
		name: "OCI/RUN",
		path: "oci/run.test",
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

func TestE2EScripts(t *testing.T) {
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

	cacheDirPriv, err := e2e.MakeTmpDir(testDir, privDirPrefix, defaultDirMode)
	if err != nil {
		t.Fatalf("failed to create privileged cache directory: %s", err)
	}

	cacheDirUnpriv, err := e2e.MakeTmpDir(testDir, unprivDirPrefix, defaultDirMode)
	if err != nil {
		t.Fatalf("failed to create unprivileged cache directory: %s", err)
	}

	sourceDir, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("failed to determine absolute source directory: %s", err)
	}

	os.Setenv("SUDO", sudo)
	os.Setenv("TESTDIR", testDir)
	os.Setenv("SINGULARITY_CACHEDIR", cacheDirUnpriv)
	os.Setenv("CACHEDIR_PRIV", cacheDirPriv)
	os.Setenv("SOURCEDIR", sourceDir)
	os.Setenv("GOOS", runtime.GOOS)
	os.Setenv("GOARCH", runtime.GOARCH)

	if testing.Verbose() {
		os.Setenv("VERBOSE_TEST", "1")

		fmt.Println("Available environment variable in test script:")
		fmt.Printf("SUDO: %s\n", sudo)
		fmt.Printf("TESTDIR: %s\n", testDir)
		fmt.Printf("SINGULARITY_CACHEDIR: %s\n", cacheDirUnpriv)
		fmt.Printf("CACHEDIR_PRIV: %s\n", cacheDirPriv)
		fmt.Printf("SOURCEDIR: %s\n", sourceDir)
		fmt.Printf("PATH: %s\n", os.Getenv("PATH"))
	} else {
		os.Setenv("VERBOSE_TEST", "0")
	}

	if err := os.Chdir("scripts"); err != nil {
		t.Fatalf("could not chdir to 'scripts' directory: %s", err)
	}

	for _, ts := range testScripts {
		stest.RunScript(nil, ts.name, ts.path, t)
	}
}

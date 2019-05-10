// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build stest

package tests

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/stest"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"

	// custom builtins
	_ "github.com/sylabs/singularity/tests/builtins/net"
	_ "github.com/sylabs/singularity/tests/builtins/tools"
)

var testScripts = []struct {
	name string
	path string
}{
	{"BUILD/BASIC", "scripts/build/basic.test"},
	{"ACTIONS/ENV", "scripts/actions/env.test"},
	{"ACTIONS/RUN", "scripts/actions/run.test"},
	{"ACTIONS/EXEC", "scripts/actions/exec.test"},
	{"ACTIONS/FROM_URI", "scripts/actions/from_uri.test"},
	{"ACTIONS/STDOUT", "scripts/actions/stdout.test"},
	{"ACTIONS/STDIN", "scripts/actions/stdin.test"},
	{"ACTIONS/PERSISTENT_OVERLAY", "scripts/actions/persistent_overlay.test"},
	{"DOCKER/PULL", "scripts/docker/pull.test"},
	{"DOCKER/DEFINITION", "scripts/docker/definition.test"},
	{"DOCKER/AUFS", "scripts/docker/aufs.test"},
	{"DOCKER/REGISTRY", "scripts/docker/registry.test"},
	{"INSTANCE/BASIC_ECHO", "scripts/instance/basic_echo.test"},
	{"INSTANCE/BASIC_OPTIONS", "scripts/instance/basic_options.test"},
	{"INSTANCE/FROM_URI", "scripts/instance/from_uri.test"},
	{"INSTANCE/CONTAIN", "scripts/instance/contain.test"},
	{"INSTANCE/CREATE_MANY", "scripts/instance/create_many.test"},
	{"OCI/BASIC", "scripts/oci/basic.test"},
}

func TestMain(t *testing.T) {
	defer os.RemoveAll(os.Getenv("TESTDIR"))

	for _, ts := range testScripts {
		stest.RunScript(ts.name, ts.path, t)
	}
}

func sudoExec(sudo string, args []string) error {
	cmd := exec.Command(sudo, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo init failed: %s", err)
	}
	return nil
}

func init() {
	useragent.InitValue(buildcfg.PACKAGE_NAME, buildcfg.PACKAGE_VERSION)

	sudo, err := exec.LookPath("sudo")
	if err != nil {
		sylog.Fatalf("sudo executable not found in $PATH")
	}

	// first sudo run to ask for password if required
	if err := sudoExec(sudo, []string{"true"}); err != nil {
		sylog.Fatalf("%s", err)
	}

	// maintain sudo session for use in test scripts without
	// password ask
	go func() {
		time.Sleep(1 * time.Minute)
		if err := sudoExec(sudo, []string{"true"}); err != nil {
			sylog.Fatalf("%s", err)
		}
	}()

	testDir, err := ioutil.TempDir("", "stest-")
	if err != nil {
		sylog.Fatalf("%s", err)
	}

	fmt.Println("Available environment variable in test script:")

	cacheDirPriv := filepath.Join(testDir, "priv")
	cacheDirUnpriv := filepath.Join(testDir, "unpriv")
	if err := os.Mkdir(cacheDirPriv, 0755); err != nil {
		sylog.Fatalf("failed to create %s: %s", cacheDirPriv, err)
	}
	if err := os.Mkdir(cacheDirUnpriv, 0755); err != nil {
		sylog.Fatalf("failed to create %s: %s", cacheDirUnpriv, err)
	}

	sourceDir := filepath.Dir(buildcfg.BUILDDIR)
	envPath := os.Getenv("PATH")

	sudoCmd := fmt.Sprintf("%s HOME=/root SINGULARITY_CACHEDIR=%s PATH=%s", sudo, cacheDirPriv, envPath)
	os.Setenv("SUDO", sudoCmd)
	fmt.Printf("SUDO: %s\n", sudo)

	os.Setenv("TESTDIR", testDir)
	fmt.Printf("TESTDIR: %s\n", testDir)

	os.Setenv("SINGULARITY_CACHEDIR", cacheDirUnpriv)
	fmt.Printf("SINGULARITY_CACHEDIR: %s\n", cacheDirUnpriv)

	os.Setenv("CACHEDIR_PRIV", cacheDirPriv)
	fmt.Printf("CACHEDIR_PRIV: %s\n", cacheDirPriv)

	os.Setenv("SOURCEDIR", sourceDir)
	fmt.Printf("SOURCEDIR: %s\n", sourceDir)

	fmt.Printf("PATH: %s\n", envPath)
}

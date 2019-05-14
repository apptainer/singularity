// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build e2e_scripts

package e2e

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
	_ "github.com/sylabs/singularity/e2e/scripts_builtins/command"
	_ "github.com/sylabs/singularity/e2e/scripts_builtins/test"
)

var testScripts = []struct {
	name string
	path string
}{
	{"BUILD/BASIC", "build/basic.test"},
	{"ACTIONS/ENV_PATH", "actions/env_path.test"},
	{"ACTIONS/RUN", "actions/run.test"},
	{"ACTIONS/EXEC", "actions/exec.test"},
	{"ACTIONS/FROM_URI", "actions/from_uri.test"},
	{"ACTIONS/STDOUT", "actions/stdout.test"},
	{"ACTIONS/STDIN", "actions/stdin.test"},
	{"ACTIONS/PERSISTENT_OVERLAY", "actions/persistent_overlay.test"},
	{"DOCKER/PULL", "docker/pull.test"},
	{"DOCKER/DEFINITION", "docker/definition.test"},
	{"DOCKER/AUFS", "docker/aufs.test"},
	{"DOCKER/REGISTRY", "docker/registry.test"},
	{"INSTANCE/BASIC_ECHO", "instance/basic_echo.test"},
	{"INSTANCE/BASIC_OPTIONS", "instance/basic_options.test"},
	{"INSTANCE/FROM_URI", "instance/from_uri.test"},
	{"INSTANCE/CONTAIN", "instance/contain.test"},
	{"INSTANCE/CREATE_MANY", "instance/create_many.test"},
	{"OCI/BASIC", "oci/basic.test"},
	{"OCI/ATTACH", "oci/attach.test"},
	{"OCI/RUN", "oci/run.test"},
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

func TestE2EScripts(t *testing.T) {
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
	// use sudo here to remove test directory in order to
	// delete image/files/directories that could be created
	// by privileged run/tests
	defer sudoExec(sudo, []string{"rm", "-rf", testDir})

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

	fmt.Println("Available environment variable in test script:")

	os.Setenv("SUDO", sudo)
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

	if err := os.Chdir("scripts"); err != nil {
		t.Fatalf("could not chdir to 'scripts' directory: %s", err)
	}

	for _, ts := range testScripts {
		stest.RunScript(ts.name, ts.path, t)
	}
}

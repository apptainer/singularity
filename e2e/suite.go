// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/sylabs/singularity/e2e/actions"

	"github.com/sylabs/singularity/e2e/docker"

	singularityenv "github.com/sylabs/singularity/e2e/env"

	"github.com/sylabs/singularity/e2e/help"

	"github.com/sylabs/singularity/e2e/imgbuild"

	"github.com/sylabs/singularity/e2e/instance"

	singularitye2e "github.com/sylabs/singularity/e2e/internal/e2e"

	"github.com/sylabs/singularity/e2e/key"

	"github.com/sylabs/singularity/e2e/pull"

	"github.com/sylabs/singularity/e2e/remote"

	"github.com/sylabs/singularity/e2e/version"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"

	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

var runDisabled = flag.Bool("run_disabled", false, "run tests that have been temporarily disabled")

// Run is the main func for the test framework, initializes the required vars
// and sets the environment for the RunE2ETests framework
func Run(t *testing.T) {
	flag.Parse()

	if *runDisabled {
		os.Setenv("E2E_RUN_DISABLED", "true")
	}
	// init buildcfg values
	useragent.InitValue(buildcfg.PACKAGE_NAME, buildcfg.PACKAGE_VERSION)

	// Ensure binary is in $PATH
	cmdPath := filepath.Join(buildcfg.BINDIR, "singularity")
	if _, err := exec.LookPath(cmdPath); err != nil {
		log.Fatalf("singularity is not installed on this system: %v", err)
	}

	os.Setenv("E2E_CMD_PATH", cmdPath)

	sysconfdir := func(fn string) string {
		return filepath.Join(buildcfg.SYSCONFDIR, "singularity", fn)
	}

	// Ensure config files are installed
	configFiles := []string{
		sysconfdir("singularity.conf"),
		sysconfdir("ecl.toml"),
		sysconfdir("capability.json"),
		sysconfdir("nvliblist.conf"),
	}

	for _, cf := range configFiles {
		if fi, err := os.Stat(cf); err != nil {
			log.Fatalf("%s is not installed on this system: %v", cf, err)
		} else if !fi.Mode().IsRegular() {
			log.Fatalf("%s is not a regular file", cf)
		} else if fi.Sys().(*syscall.Stat_t).Uid != 0 {
			log.Fatalf("%s must be owned by root", cf)
		}
	}

	// Make temp dir for tests
	name, err := ioutil.TempDir("", "stest.")
	if err != nil {
		log.Fatalf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(name)
	if err := os.Chmod(name, 0777); err != nil {
		log.Fatalf("failed to chmod temporary directory: %v", err)
	}
	os.Setenv("E2E_TEST_DIR", name)

	// Build a base image for tests
	imagePath := path.Join(name, "test.sif")
	t.Log(imagePath)
	os.Setenv("E2E_IMAGE_PATH", imagePath)
	defer os.Remove(imagePath)

	// Start registry for tests
	singularitye2e.PrepRegistry(t)
	defer singularitye2e.KillRegistry(t)

	// RunE2ETests by functionality

	t.Run("KEYCMD", key.RunE2ETests)

	t.Run("BUILD", imgbuild.RunE2ETests)

	t.Run("ACTIONS", actions.RunE2ETests)

	t.Run("DOCKER", docker.RunE2ETests)

	t.Run("PULL", pull.RunE2ETests)

	t.Run("REMOTE", remote.RunE2ETests)

	t.Run("INSTANCE", instance.RunE2ETests)

	t.Run("HELP", help.RunE2ETests)

	t.Run("ENV", singularityenv.RunE2ETests)

	t.Run("VERSION", version.RunE2ETests)

}

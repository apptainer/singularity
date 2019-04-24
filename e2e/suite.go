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

//	"github.com/sylabs/singularity/e2e/actions"
	"github.com/sylabs/singularity/e2e/docker"
	"github.com/sylabs/singularity/e2e/imgbuild"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

var runDisabled = flag.Bool("run_disabled", false, "run tests that have been temporarily disabled")

// Run is the main func for the test framework, initializes the required vars
// and set the environment for the RunE2ETests framework
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
	opts := imgbuild.Opts{
		Force:   true,
		Sandbox: false,
	}
	if b, err := imgbuild.ImageBuild(cmdPath, opts, imagePath, "./testdata/Singularity"); err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}
	os.Setenv("E2E_IMAGE_PATH", imagePath)
	defer os.Remove(imagePath)

	// RunE2ETests by functionality
//	t.Run("BUILD", imgbuild.RunE2ETests)
//	t.Run("ACTIONS", actions.RunE2ETests)
	t.Run("DOCKER", docker.RunE2ETests)
}

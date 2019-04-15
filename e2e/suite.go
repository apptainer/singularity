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
	"syscall"
	"testing"

	"github.com/sylabs/singularity/e2e/actions"
	"github.com/sylabs/singularity/e2e/imgbuild"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
)

var runDisabled = flag.Bool("run_disabled", false, "run tests that have been temporarily disabled")

// Run is the main func for the test framework, initializes the required vars
// and set the environment for the RunE2ETests framework
func Run(t *testing.T) {
	// Ensure binary is in $PATH
	cmdPath, err := exec.LookPath("singularity")
	if err != nil {
		log.Fatalf("singularity is not installed on this system: %v", err)
	}
	os.Setenv("E2E_CMD_PATH", cmdPath)

	// Ensure config files are installed
	configFiles := []struct {
		fileName string
		filePath string
	}{
		{"singularity.conf", "/singularity/singularity.conf"},
		{"ecl.toml", "/singularity/ecl.toml"},
		{"capability.json", "/singularity/capability.json"},
		{"nvliblist.conf", "/singularity/nvliblist.conf"},
	}
	for _, cf := range configFiles {
		if fi, err := os.Stat(buildcfg.SYSCONFDIR + cf.filePath); err != nil {
			log.Fatalf("%s is not installed on this system: %v", cf.fileName, err)
		} else if !fi.Mode().IsRegular() {
			log.Fatalf("%s is not a regular file", cf.fileName)
		} else if fi.Sys().(*syscall.Stat_t).Uid != 0 {
			log.Fatalf("%s must be owned by root", cf.fileName)
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
	t.Run("BUILD", imgbuild.RunE2ETests)
	t.Run("ACTIONS", actions.RunE2ETests)
}

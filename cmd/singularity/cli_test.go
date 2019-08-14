// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"syscall"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/test"
)

var (
	cmdPath string
	testDir string
)

var runDisabled = flag.Bool("run_disabled", false, "run tests that have been temporarily disabled")

func TestSelfTest(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// We always prefer to run tests with a clean temporary image cache rather
	// than using the cache of the user running the test.
	// In order to unit test using the singularity cli that is thread-safe,
	// we prepare a temporary cache that the process running the command will
	// use.
	cmd := exec.Command(cmdPath, "selftest")
	setupCmdCache(t, cmd, "image-cache")
	if b, err := cmd.CombinedOutput(); err == nil {
		t.Log(string(b))
		t.Fatal("unexpected success running selftest")
	}
}

func run(m *testing.M) int {

	// Ensure binary is in $PATH
	path, err := exec.LookPath("singularity")
	if err != nil {
		log.Fatalf("singularity is not installed on this system: %v", err)
	}
	cmdPath = path

	// Ensure config is installed
	if fi, err := os.Stat(buildcfg.SINGULARITY_CONF_FILE); err != nil {
		log.Fatalf("singularity config is not installed on this system: %v", err)
	} else if !fi.Mode().IsRegular() {
		log.Fatalf("singularity config is not a regular file")
	} else if fi.Sys().(*syscall.Stat_t).Uid != 0 {
		log.Fatalf("singularity.conf must be owned by root")
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
	testDir = name

	return m.Run()
}

func TestMain(m *testing.M) {
	flag.Parse()

	os.Exit(run(m))
}

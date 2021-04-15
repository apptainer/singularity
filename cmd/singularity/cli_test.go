// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build integration_test

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
)

var (
	cmdPath string
	testDir string
)

var runDisabled = flag.Bool("run_disabled", false, "run tests that have been temporarily disabled")

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

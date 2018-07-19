// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"log"
	"os"
	"os/exec"
	"testing"

	"github.com/singularityware/singularity/src/pkg/buildcfg"
)

var cmdPath string

func TestSelfTest(t *testing.T) {
	cmd := exec.Command(cmdPath, "selftest")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure running selftest: %v", err)
	}
}

func TestMain(m *testing.M) {

	// Sanity checks for sudo
	if b, err := exec.Command("sudo", "true").CombinedOutput(); err != nil {
		log.Print(string(b))
		log.Fatalf("unexpected failure running 'sudo true': %v", err)
	}
	if b, err := exec.Command("sudo", "false").CombinedOutput(); err == nil {
		log.Print(string(b))
		log.Fatalf("unexpected success running 'sudo false'")
	}

	// Ensure binary is in $PATH
	path, err := exec.LookPath("singularity")
	if err != nil {
		log.Fatalf("singularity is not installed on this system: %v", err)
	}
	cmdPath = path

	// Ensure config is installed
	if fi, err := os.Stat(buildcfg.SYSCONFDIR + "/singularity/singularity.conf"); err != nil {
		log.Fatalf("singularity config is not installed on this system: %v", err)
	} else if !fi.Mode().IsRegular() {
		log.Fatalf("singularity config is not a regular file")
	}

	os.Exit(m.Run())
}

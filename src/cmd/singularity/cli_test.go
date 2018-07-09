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

func run(m *testing.M) int {

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

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

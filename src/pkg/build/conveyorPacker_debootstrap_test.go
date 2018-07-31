// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"os"
	"os/exec"
	"testing"

	"github.com/singularityware/singularity/src/pkg/test"
)

func TestDebootstrapConveyor(t *testing.T) {

	if testing.Short() {
		t.SkipNow()
	}

	if _, err := exec.LookPath("debootstrap"); err != nil {
		t.Skip("skipping test, debootstrap not installed")
	}

	test.EnsurePrivilege(t)

	testDef := Definition{}

	testDef.Header = map[string]string{
		"bootstrap": "debootstrap",
		"osversion": "bionic",
		"mirrorurl": "http://us.archive.ubuntu.com/ubuntu/",
		"include":   "apt python ",
	}

	dc := DebootstrapConveyor{}

	err := dc.Get(testDef)
	//clean up tmpfs since assembler isnt called
	defer os.RemoveAll(dc.tmpfs)
	if err != nil {
		t.Fatalf("Debootstrap Get failed: %v", err)
	}
}

func TestDebootstrapPacker(t *testing.T) {

	if _, err := exec.LookPath("debootstrap"); err != nil {
		t.Skip("skipping test, debootstrap not installed")
	}

	test.EnsurePrivilege(t)

	testDef := Definition{}

	testDef.Header = map[string]string{
		"bootstrap": "debootstrap",
		"osversion": "bionic",
		"mirrorurl": "http://us.archive.ubuntu.com/ubuntu/",
		"include":   "apt python ",
	}

	dcp := DebootstrapConveyorPacker{}

	err := dcp.Get(testDef)
	//clean up tmpfs since assembler isnt called
	defer os.RemoveAll(dcp.tmpfs)
	if err != nil {
		t.Fatalf("Debootstrap Get failed: %v", err)
	}

	_, err = dcp.Pack()
	if err != nil {
		t.Fatalf("Debootstrap Pack failed: %v", err)
	}
}

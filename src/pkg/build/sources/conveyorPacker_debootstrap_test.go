// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources_test

import (
	"os/exec"
	"testing"

	"github.com/singularityware/singularity/src/pkg/build/sources"
	"github.com/singularityware/singularity/src/pkg/build/types"
	"github.com/singularityware/singularity/src/pkg/test"
)

func TestDebootstrapConveyor(t *testing.T) {

	if _, err := exec.LookPath("debootstrap"); err != nil {
		t.Skip("skipping test, debootstrap not installed")
	}

	test.EnsurePrivilege(t)

	testDef := types.Definition{}

	testDef.Header = map[string]string{
		"bootstrap": "debootstrap",
		"osversion": "bionic",
		"mirrorurl": "http://us.archive.ubuntu.com/ubuntu/",
		"include":   "apt python ",
	}

	dc := sources.DebootstrapConveyor{}

	err := dc.Get(testDef)
	if err != nil {
		t.Fatalf("Debootstrap Get failed: %v", err)
	}
}

func TestDebootstrapPacker(t *testing.T) {

	if _, err := exec.LookPath("debootstrap"); err != nil {
		t.Skip("skipping test, debootstrap not installed")
	}

	test.EnsurePrivilege(t)

	testDef := types.Definition{}

	testDef.Header = map[string]string{
		"Bootstrap": "debootstrap",
		"OSVersion": "bionic",
		"MirrorURL": "http://us.archive.ubuntu.com/ubuntu/",
		"Include":   "apt python ",
	}

	dcp := sources.DebootstrapConveyorPacker{}

	err := dcp.Get(testDef)
	if err != nil {
		t.Fatalf("Debootstrap Get failed: %v", err)
	}

	_, err = dcp.Pack()
	if err != nil {
		t.Fatalf("Debootstrap Pack failed: %v", err)
	}
}

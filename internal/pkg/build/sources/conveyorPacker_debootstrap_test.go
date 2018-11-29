// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources_test

import (
	"os/exec"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/build/sources"
	"github.com/sylabs/singularity/internal/pkg/build/types"
	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestDebootstrapConveyor(t *testing.T) {

	if testing.Short() {
		t.SkipNow()
	}

	if _, err := exec.LookPath("debootstrap"); err != nil {
		t.Skip("skipping test, debootstrap not installed")
	}

	test.EnsurePrivilege(t)

	b, err := types.NewBundle("", "sbuild-debootstrap")
	if err != nil {
		return
	}

	b.Recipe.Header = map[string]string{
		"bootstrap": "debootstrap",
		"osversion": "bionic",
		"mirrorurl": "http://us.archive.ubuntu.com/ubuntu/",
		"include":   "apt python ",
	}

	cp := sources.DebootstrapConveyorPacker{}

	err = cp.Get(b)
	// clean up tmpfs since assembler isnt called
	defer cp.CleanUp()
	if err != nil {
		t.Fatalf("Debootstrap Get failed: %v", err)
	}
}

func TestDebootstrapPacker(t *testing.T) {

	if _, err := exec.LookPath("debootstrap"); err != nil {
		t.Skip("skipping test, debootstrap not installed")
	}

	test.EnsurePrivilege(t)

	b, err := types.NewBundle("", "sbuild-debootstrap")
	if err != nil {
		return
	}

	b.Recipe.Header = map[string]string{
		"bootstrap": "debootstrap",
		"osversion": "bionic",
		"mirrorurl": "http://us.archive.ubuntu.com/ubuntu/",
		"include":   "apt python ",
	}

	cp := sources.DebootstrapConveyorPacker{}

	err = cp.Get(b)
	// clean up tmpfs since assembler isnt called
	defer cp.CleanUp()
	if err != nil {
		t.Fatalf("Debootstrap Get failed: %v", err)
	}

	_, err = cp.Pack()
	if err != nil {
		t.Fatalf("Debootstrap Pack failed: %v", err)
	}
}

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

const archDef = "./testdata_good/arch/arch"

func TestArchConveyor(t *testing.T) {

	if testing.Short() {
		t.SkipNow()
	}

	if _, err := exec.LookPath("pacstrap"); err != nil {
		t.Skip("skipping test, pacstrap not installed")
	}

	test.EnsurePrivilege(t)

	defFile, err := os.Open(archDef)
	if err != nil {
		t.Fatalf("unable to open file %s: %v\n", archDef, err)
	}
	defer defFile.Close()

	def, err := ParseDefinitionFile(defFile)
	if err != nil {
		t.Fatalf("failed to parse definition file %s: %v\n", archDef, err)
	}

	ac := &ArchConveyor{}

	err = ac.Get(def)
	//clean up tmpfs since assembler isnt called
	defer os.RemoveAll(ac.tmpfs)
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", archDef, err)
	}
}

func TestArchPacker(t *testing.T) {

	if _, err := exec.LookPath("pacstrap"); err != nil {
		t.Skip("skipping test, pacstrap not installed")
	}

	test.EnsurePrivilege(t)

	defFile, err := os.Open(archDef)
	if err != nil {
		t.Fatalf("unable to open file %s: %v\n", archDef, err)
	}
	defer defFile.Close()

	def, err := ParseDefinitionFile(defFile)
	if err != nil {
		t.Fatalf("failed to parse definition file %s: %v\n", archDef, err)
	}

	acp := &ArchConveyorPacker{}

	err = acp.Get(def)
	//clean up tmpfs since assembler isnt called
	defer os.RemoveAll(acp.tmpfs)
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", archDef, err)
	}

	_, err = acp.Pack()
	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", archDef, err)
	}
}

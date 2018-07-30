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

	if err := ac.Get(def); err != nil {
		//clean up tmpfs since assembler isnt called
		os.RemoveAll(ac.tmpfs)
		t.Fatalf("failed to Get from %s: %v\n", archDef, err)
	}
	//clean up tmpfs since assembler isnt called
	os.RemoveAll(ac.tmpfs)
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

	if err := acp.Get(def); err != nil {
		//clean up tmpfs since assembler isnt called
		os.RemoveAll(acp.tmpfs)
		t.Fatalf("failed to Get from %s: %v\n", archDef, err)
	}
	//clean up tmpfs since assembler isnt called
	os.RemoveAll(acp.tmpfs)

	_, err = acp.Pack()
	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", archDef, err)
	}
}

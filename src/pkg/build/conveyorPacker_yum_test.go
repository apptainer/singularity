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

const yumDef = "./testdata_good/yum/yum"

func TestYumConveyor(t *testing.T) {

	if _, err := exec.LookPath("yum"); err != nil {
		t.Skip("skipping test, yum not installed")
	}

	test.EnsurePrivilege(t)

	defFile, err := os.Open(yumDef)
	if err != nil {
		t.Fatalf("unable to open file %s: %v\n", yumDef, err)
	}
	defer defFile.Close()

	def, err := ParseDefinitionFile(defFile)
	if err != nil {
		t.Fatalf("failed to parse definition file %s: %v\n", yumDef, err)
	}

	yc := &YumConveyor{}

	if err := yc.Get(def); err != nil {
		//clean up bundle since assembler isnt called
		os.RemoveAll(yc.b.Path)
		t.Fatalf("failed to Get from %s: %v\n", yumDef, err)
	}
	//clean up tmpfs since assembler isnt called
	os.RemoveAll(yc.b.Path)
}

func TestYumPacker(t *testing.T) {

	if _, err := exec.LookPath("yum"); err != nil {
		t.Skip("skipping test, yum not installed")
	}

	test.EnsurePrivilege(t)

	defFile, err := os.Open(yumDef)
	if err != nil {
		t.Fatalf("unable to open file %s: %v\n", yumDef, err)
	}
	defer defFile.Close()

	def, err := ParseDefinitionFile(defFile)
	if err != nil {
		t.Fatalf("failed to parse definition file %s: %v\n", yumDef, err)
	}

	ycp := &YumConveyorPacker{}

	if err := ycp.Get(def); err != nil {
		//clean up tmpfs since assembler isnt called
		os.RemoveAll(ycp.b.Path)
		t.Fatalf("failed to Get from %s: %v\n", yumDef, err)
	}
	//clean up tmpfs since assembler isnt called
	os.RemoveAll(ycp.b.Path)

	_, err = ycp.Pack()
	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", yumDef, err)
	}
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"os"
	"testing"

	"github.com/singularityware/singularity/src/pkg/test"
)

const busyBoxDef = "./testdata_good/busybox/busybox"

func TestBusyBoxConveyor(t *testing.T) {

	if testing.Short() {
		t.SkipNow()
	}

	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	defFile, err := os.Open(busyBoxDef)
	if err != nil {
		t.Fatalf("unable to open file %s: %v\n", busyBoxDef, err)
	}
	defer defFile.Close()

	def, err := ParseDefinitionFile(defFile)
	if err != nil {
		t.Fatalf("failed to parse definition file %s: %v\n", busyBoxDef, err)
	}

	bc := &BusyBoxConveyor{}

	if err := bc.Get(def); err != nil {
		//clean up tmpfs since assembler isnt called
		os.RemoveAll(bc.tmpfs)
		t.Fatalf("failed to Get from %s: %v\n", busyBoxDef, err)
	}
	//clean up tmpfs since assembler isnt called
	os.RemoveAll(bc.tmpfs)

}

func TestBusyBoxPacker(t *testing.T) {

	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	defFile, err := os.Open(busyBoxDef)
	if err != nil {
		t.Fatalf("unable to open file %s: %v\n", busyBoxDef, err)
	}
	defer defFile.Close()

	def, err := ParseDefinitionFile(defFile)
	if err != nil {
		t.Fatalf("failed to parse definition file %s: %v\n", busyBoxDef, err)
	}

	bcp := &BusyBoxConveyorPacker{}

	if err := bcp.Get(def); err != nil {
		//clean up tmpfs since assembler isnt called
		os.RemoveAll(bcp.tmpfs)
		t.Fatalf("failed to Get from %s: %v\n", busyBoxDef, err)
	}

	//clean up tmpfs since assembler isnt called
	defer os.RemoveAll(bcp.tmpfs)

	_, err = bcp.Pack()
	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", busyBoxDef, err)
	}
}

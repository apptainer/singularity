// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"os"
	"os/exec"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/build/types"
	"github.com/sylabs/singularity/internal/pkg/build/types/parser"
	"github.com/sylabs/singularity/internal/pkg/test"
)

const yumDef = "../testdata_good/yum/yum"

func TestYumConveyor(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	_, dnfErr := exec.LookPath("dnf")
	_, yumErr := exec.LookPath("yum")
	if dnfErr != nil && yumErr != nil {
		t.Skip("skipping test, neither dnf nor yum found")
	}

	test.EnsurePrivilege(t)

	defFile, err := os.Open(yumDef)
	if err != nil {
		t.Fatalf("unable to open file %s: %v\n", yumDef, err)
	}
	defer defFile.Close()

	// create bundle to build into
	b, err := types.NewBundle("", "sbuild-yum")
	if err != nil {
		return
	}

	b.Recipe, err = parser.ParseDefinitionFile(defFile)
	if err != nil {
		t.Fatalf("failed to parse definition file %s: %v\n", yumDef, err)
	}

	yc := &YumConveyor{}

	err = yc.Get(b)
	// clean up bundle since assembler isnt called
	defer os.RemoveAll(yc.b.Path)
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", yumDef, err)
	}
}

func TestYumPacker(t *testing.T) {
	_, dnfErr := exec.LookPath("dnf")
	_, yumErr := exec.LookPath("yum")
	if dnfErr != nil && yumErr != nil {
		t.Skip("skipping test, neither dnf nor yum found")
	}

	test.EnsurePrivilege(t)

	defFile, err := os.Open(yumDef)
	if err != nil {
		t.Fatalf("unable to open file %s: %v\n", yumDef, err)
	}
	defer defFile.Close()

	// create bundle to build into
	b, err := types.NewBundle("", "sbuild-yum")
	if err != nil {
		return
	}

	b.Recipe, err = parser.ParseDefinitionFile(defFile)
	if err != nil {
		t.Fatalf("failed to parse definition file %s: %v\n", yumDef, err)
	}

	ycp := &YumConveyorPacker{}

	err = ycp.Get(b)
	// clean up tmpfs since assembler isnt called
	defer os.RemoveAll(ycp.b.Path)
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", yumDef, err)
	}

	_, err = ycp.Pack()
	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", yumDef, err)
	}
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/build/types/parser"
)

const yumDef = "../../../../examples/centos/Singularity"

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
	b, err := types.NewBundle(filepath.Join(os.TempDir(), "sbuild-yum"), os.TempDir())
	if err != nil {
		return
	}

	b.Recipe, err = parser.ParseDefinitionFile(defFile)
	if err != nil {
		t.Fatalf("failed to parse definition file %s: %v\n", yumDef, err)
	}

	yc := &YumConveyor{}

	err = yc.Get(context.Background(), b)
	// clean up bundle since assembler isnt called
	defer yc.b.Remove()
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", yumDef, err)
	}
}

func TestYumPacker(t *testing.T) {
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
	b, err := types.NewBundle(filepath.Join(os.TempDir(), "sbuild-yum"), os.TempDir())
	if err != nil {
		return
	}

	b.Recipe, err = parser.ParseDefinitionFile(defFile)
	if err != nil {
		t.Fatalf("failed to parse definition file %s: %v\n", yumDef, err)
	}

	ycp := &YumConveyorPacker{}

	err = ycp.Get(context.Background(), b)
	// clean up tmpfs since assembler isnt called
	defer ycp.b.Remove()
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", yumDef, err)
	}

	_, err = ycp.Pack(context.Background())
	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", yumDef, err)
	}
}

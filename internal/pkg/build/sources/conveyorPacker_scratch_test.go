// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources_test

import (
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/build/sources"
	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/build/types/parser"
)

const scratchDef = "../../../../pkg/build/types/parser/testdata_good/scratch/scratch"

func TestScratchConveyor(t *testing.T) {

	if testing.Short() {
		t.SkipNow()
	}

	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	defFile, err := os.Open(scratchDef)
	if err != nil {
		t.Fatalf("unable to open file %s: %v\n", scratchDef, err)
	}
	defer defFile.Close()

	b, err := types.NewBundle("", "sbuild-scratch")
	if err != nil {
		return
	}

	b.Recipe, err = parser.ParseDefinitionFile(defFile)
	if err != nil {
		t.Fatalf("failed to parse definition file %s: %v\n", scratchDef, err)
	}

	c := &sources.ScratchConveyor{}

	err = c.Get(b)
	// clean up tmpfs since assembler isnt called
	defer c.CleanUp()
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", scratchDef, err)
	}
}

func TestScratchPacker(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	defFile, err := os.Open(scratchDef)
	if err != nil {
		t.Fatalf("unable to open file %s: %v\n", scratchDef, err)
	}
	defer defFile.Close()

	b, err := types.NewBundle("", "sbuild-scratch")
	if err != nil {
		return
	}

	b.Recipe, err = parser.ParseDefinitionFile(defFile)
	if err != nil {
		t.Fatalf("failed to parse definition file %s: %v\n", scratchDef, err)
	}

	cp := &sources.ScratchConveyorPacker{}

	err = cp.Get(b)
	// clean up tmpfs since assembler isnt called
	defer cp.CleanUp()
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", scratchDef, err)
	}

	_, err = cp.Pack()
	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", scratchDef, err)
	}
}

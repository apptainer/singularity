// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
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

var zyppDef = [...]string{
	"../../../../examples/opensuse/Singularity",
	"../../../../examples/sle/Singularity",
}

func testForSLE(t *testing.T, b *types.Bundle) {
	if _, ok := b.Recipe.Header["product"]; ok {
		if _, err := exec.LookPath("SUSEConnect"); err != nil {
			t.Skip("skipping test, SUSEConnect not found")
		}
		user := b.Recipe.Header["user"]
		regcode := b.Recipe.Header["user"]
		if user == "" || regcode == "" {
			t.Skip("skipping test: specify valid SLE user and regcode")
		}
	}
}

func TestZypperConveyor(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	test.EnsurePrivilege(t)

	if _, err := exec.LookPath("zypper"); err != nil {
		t.Skip("skipping test, zypper not found")
	}

	for _, defName := range zyppDef {
		defFile, err := os.Open(defName)
		if err != nil {
			t.Fatalf("unable to open file %s: %v\n", defName, err)
		}
		defer defFile.Close()

		// create bundle to build into
		b, err := types.NewBundle(filepath.Join(os.TempDir(), "sbuild-zypper"), os.TempDir())
		if err != nil {
			return
		}

		b.Recipe, err = parser.ParseDefinitionFile(defFile)
		if err != nil {
			t.Fatalf("failed to parse definition file %s: %v\n", defName, err)
		}

		testForSLE(t, b)

		zc := &ZypperConveyorPacker{}

		err = zc.Get(context.Background(), b)
		// clean up tmpfs since assembler isnt called
		defer zc.b.Remove()
		if err != nil {
			t.Fatalf("failed to Get from %s: %v\n", defName, err)
		}
	}
}

func TestZypperPacker(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	test.EnsurePrivilege(t)

	if _, err := exec.LookPath("zypper"); err != nil {
		t.Skip("skipping test, zypper not found")
	}

	for _, defName := range zyppDef {
		defFile, err := os.Open(defName)
		if err != nil {
			t.Fatalf("unable to open file %s: %v\n", defName, err)
		}
		defer defFile.Close()

		// create bundle to build into
		b, err := types.NewBundle(filepath.Join(os.TempDir(), "sbuild-zypper"), os.TempDir())
		if err != nil {
			return
		}

		b.Recipe, err = parser.ParseDefinitionFile(defFile)
		if err != nil {
			t.Fatalf("failed to parse definition file %s: %v\n", defName, err)
		}

		testForSLE(t, b)

		zcp := &ZypperConveyorPacker{}

		err = zcp.Get(context.Background(), b)
		// clean up tmpfs since assembler isnt called
		defer zcp.b.Remove()
		if err != nil {
			t.Fatalf("failed to Get from %s: %v\n", defName, err)
		}

		_, err = zcp.Pack(context.Background())
		if err != nil {
			t.Fatalf("failed to Pack from %s: %v\n", defName, err)
		}
	}
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources_test

import (
	"testing"

	"github.com/sylabs/singularity/internal/pkg/build/sources"
	"github.com/sylabs/singularity/internal/pkg/build/types"
	"github.com/sylabs/singularity/internal/pkg/test"
)

const (
	shubURI = "shub://ikaneshiro/singularityhub:latest"
)

// TestShubConveyor tests if we can pull an image from singularity hub
func TestShubConveyor(t *testing.T) {

	if testing.Short() {
		t.SkipNow()
	}

	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	b, err := types.NewBundle("", "sbuild-shub")
	if err != nil {
		return
	}

	b.Recipe, err = types.NewDefinitionFromURI(shubURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", shubURI, err)
	}

	cp := &sources.ShubConveyorPacker{}

	err = cp.Get(b)
	// clean up tmpfs since assembler isnt called
	defer cp.CleanUp()
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", shubURI, err)
	}
}

// TestShubPacker checks if we can create a Bundle from the pulled image
func TestShubPacker(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	b, err := types.NewBundle("", "sbuild-shub")
	if err != nil {
		return
	}

	b.Recipe, err = types.NewDefinitionFromURI(shubURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", shubURI, err)
	}

	scp := &sources.ShubConveyorPacker{}

	err = scp.Get(b)
	// clean up tmpfs since assembler isnt called
	defer scp.CleanUp()
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", shubURI, err)
	}

	_, err = scp.Pack()
	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", shubURI, err)
	}
}

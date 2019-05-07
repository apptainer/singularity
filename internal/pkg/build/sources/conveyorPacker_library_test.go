// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources_test

import (
	"testing"

	"github.com/sylabs/singularity/internal/pkg/build/sources"
	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/pkg/build/types"
)

const (
	libraryURL = "https://library.sylabs.io/"
	libraryURI = "library://alpine:latest"
)

// TestLibraryConveyor tests if we can pull an image from singularity hub
func TestLibraryConveyor(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	test.EnsurePrivilege(t)

	b, err := types.NewBundle("", "sbuild-library")
	if err != nil {
		return
	}

	b.Opts.LibraryURL = libraryURL

	b.Recipe, err = types.NewDefinitionFromURI(libraryURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", libraryURI, err)
	}

	cp := &sources.LibraryConveyorPacker{}

	err = cp.Get(b)
	// clean up tmpfs since assembler isnt called
	defer cp.CleanUp()
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", libraryURI, err)
	}
}

func createBundle(t *testing.T) *types.Bundle {
	b, err := types.NewBundle("", "sbuild-library")
	if err != nil {
		//t.Fatalf("failed to create new bundle: %s", err)
		return nil
	}

	b.Opts.LibraryURL = libraryURL

	b.Recipe, err = types.NewDefinitionFromURI(libraryURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", libraryURI, err)
	}

	return b
}

// TestLibraryPacker checks if we can create a Bundle from the pulled image
func TestLibraryPacker(t *testing.T) {
	test.EnsurePrivilege(t)

	tempCacheConfig := test.CacheTestInit(t)
	defer test.CacheTestFinalize(t, tempCacheConfig)

	cp := &sources.LibraryConveyorPacker{}
	b := createBundle(t)
	err := cp.Get(b)
	// clean up tmpfs since assembler isnt called
	defer cp.CleanUp()
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", libraryURI, err)
	}

	_, err = cp.Pack()
	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", libraryURI, err)
	}
}

// TestGet focuses on error cases for the Get() function
func TestGet(t *testing.T) {
	test.EnsurePrivilege(t)

	// We create a valid cache but we will alter it to create a specific
	// invalid configuration
	tempCacheConfig := test.CacheTestInit(t)
	defer test.CacheTestFinalize(t, tempCacheConfig)

	err := test.CacheTestInvalidate(t, tempCacheConfig)
	if err != nil {
		t.Fatalf("failed to invalidate cache: %s", err)
	}

	cp := &sources.LibraryConveyorPacker{}
	b := createBundle(t)

	err = cp.Get(b)
	if err == nil {
		t.Fatal("successfully packed data with an invalid cache")
	}
	defer cp.CleanUp()
}

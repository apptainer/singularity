// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources_test

import (
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/build/sources"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
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

	cacheDir := test.SetCacheDir(t, "")
	defer test.CleanCacheDir(t, cacheDir)

	err := os.Setenv(cache.DirEnv, cacheDir)
	if err != nil {
		t.Fatalf("failed to set %s environment variable: %s", cache.DirEnv, err)
	}

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

// TestLibraryPacker checks if we can create a Bundle from the pulled image
func TestLibraryPacker(t *testing.T) {
	test.EnsurePrivilege(t)
	cacheDir := test.SetCacheDir(t, "")
	defer test.CleanCacheDir(t, cacheDir)

	err := os.Setenv(cache.DirEnv, cacheDir)
	if err != nil {
		t.Fatalf("failed to set %s environment variable: %s", cache.DirEnv, err)
	}

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

	_, err = cp.Pack()
	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", libraryURI, err)
	}
}

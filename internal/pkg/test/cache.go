// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"testing"
)

// SetCacheDir creates a new image cache in the context of testing
// and set the appropriate environment variable to ensure that the
// temporary cache is used by the caller.
func SetCacheDir(t *testing.T) {
	basedir := os.Getenv("SINGULARITY_TESTBASEDIR")

	if basedir != "" {
		// We make sure the base directory for the test is created.
		// we rely only on package from Golang to avoid dependency cycles
		oldmask := syscall.Umask(0)
		defer syscall.Umask(oldmask)
		err := os.MkdirAll(basedir, 0755)
		// This is an error only outside of the case where the directory already exists
		if err != nil && !os.IsExist(err) {
			t.Fatalf("failed to create test base directory %s: %s", basedir, err)
		}
	} else {
		t.Fatal("SINGULARITY_TESTBASEDIR not set")
	}

	// We create a unique temporary directory for the image cache since Go run
	// tests concurrently and since SINGULARITY_TESTBASEDIR is a static directory.
	dir, err := ioutil.TempDir(basedir, "image_cache-")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}

	// We do not use cache.DirEnv to avoid a dependency cycle
	err = os.Setenv("SINGULARITY_CACHEDIR", dir)
	if err != nil {
		t.Fatalf("failed to set SINGULARITY_CACHEDIR")
	}
}

// CleanCacheDir deleted the temporary cache that was created for
// testing purposes.
func CleanCacheDir(t *testing.T) {
	// We do not use cache.DirEnv to avoid a dependency cycle
	cacheDir := os.Getenv("SINGULARITY_CACHEDIR")

	// We clean the image cache only if we are not using the
	// default cache; we do not want to clean the developer's
	// default cache.
	if cacheDir != "" {
		os.RemoveAll(cacheDir)
	} else {
		fmt.Println("WARNING! the test is using the default image cache")
	}
}

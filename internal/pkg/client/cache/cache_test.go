// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

/*
A set of functions internal to the package for testing.
Public helper functions for testinng are in 'testing.go'
*/

// Constants used throughout the tests
const (
	validSHASum   = "0"
	invalidSHASum = "" //"not a SHA sum"
	validPath     = "a_dummy_image"
	invalidPath   = ""
	cacheCustom   = "/tmp/customcachedir"
)

// createTempCache creates a valid Singularity cache in a temporary directory
// to ease testing
func createTempCache(t *testing.T) *SingularityCache {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal("Impossible to create temporary directory")
	}

	newCache, err := Init(dir)
	if newCache == nil || err != nil {
		t.Fatal("cannot create temporary cache")
	}

	return newCache
}

// setupCache abstracts the creation of a new cache, mainly all the associated
// error checking
func setupCache(t *testing.T) *SingularityCache {
	newCache, err := Create()
	if err != nil {
		return nil
	}
	return newCache
}

// cleanupCache will free/destroy a cache ONLY if it is NOT the default cache.
// The goal here is not interfer with the default cache that is populating
// while using Singularity when running tests. Used in conjunction with setupCache().
func cleanupCache(t *testing.T, c *SingularityCache) {
	if c.Default == false {
		c.Clean()
	}

	// We restore the previous value of DirEnv
	os.Setenv(DirEnv, c.PreviousDirEnv)
}

// getDefaultCacheValues is a helper function that returns the typical
// expected values when creating a temporary cache. This mainly aims at
// avoiding code duplication and abstract the location of the cache.
func getDefaultCacheValues(t *testing.T) (string, string) {
	me, err := user.Current() //test.GetCurrentUser(t)
	if me == nil || err != nil {
		t.Fatal("failed getting the current user")
	}

	expectedDefaultCache := filepath.Join(me.HomeDir, ".singularity", "cache")
	expectedCustomCache := filepath.Join(cacheCustom, "cache")

	return expectedDefaultCache, expectedCustomCache
}

// createFakeImage allocates the minimum resources required to simulate a
// valid image in the context of cache testing.
func createFakeImage(t *testing.T, base string) {
	err := os.MkdirAll(filepath.Join(base, validSHASum), 0755)
	if err != nil {
		t.Fatalf("cannot create directory %s: %s\n", filepath.Join(base, validSHASum), err)
	}
	validImage := filepath.Join(base, validSHASum, validPath)
	_, err = os.Create(validImage) // no need to explicitly delete the file, it will be when cleaning the cache
	if err != nil {
		t.Fatalf("cannot create file %s: %s\n", validImage, err)
	}
}

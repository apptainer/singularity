// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
)

// TempCache is a structure used as an opaque handle that stores all the
// data specific to a given test with a temporary Singularity cache
type TempCache struct {
	PreviousDirEnv  string
	BaseDir         string
	previousBaseDir string
}

// CacheTestInit performs all the under the cover tasks to setup a new temporary
// cache for testing. The function returns a structure that can be later on
// used for cleanup
func CacheTestInit(t *testing.T) *TempCache {
	c := new(TempCache)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("cannot create temporary cache")
	}
	c.PreviousDirEnv = os.Getenv(cache.DirEnv)
	c.BaseDir = dir
	err = os.Setenv(cache.DirEnv, dir)
	if err != nil {
		t.Fatalf("cannot set env variable while setting up a temporary cache")
	}

	return c
}

// CacheTestInvalidate modifies an existing cache to make it invalidate by
// setting the base directory to a file instead of a directory.
func CacheTestInvalidate(t *testing.T, c *TempCache) error {
	if c == nil {
		return fmt.Errorf("invalid cache configuration")
	}

	file, err := ioutil.TempFile("", "")
	// The code path for the error case is not easily testable to instead of
	// returning an error, we simply kill the test
	if err != nil {
		t.Fatalf("failed to create temporary file: %s", err)
	}
	path := file.Name()
	file.Close()

	c.previousBaseDir = c.BaseDir
	c.BaseDir = path
	err = os.Setenv(cache.DirEnv, path)
	// The code path for the error case is not easily testable to instead of
	// returning an error, we simply kill the test
	if err != nil {
		t.Fatalf("failed to set environment variable: %s", err)
	}

	return nil
}

// CacheTestFinalize cleans up the environment created when calling CacheTestInit()
func CacheTestFinalize(t *testing.T, c *TempCache) error {
	if c == nil {
		return fmt.Errorf("undefined cache test configuration")
	}

	err := os.RemoveAll(c.BaseDir)
	// In case of error, do not fail, we want to clean up as much as possible
	if err != nil {
		fmt.Printf("WARNING: cannot remove %s: %s\n", c.BaseDir, err)
	}

	err = os.Setenv(cache.DirEnv, c.PreviousDirEnv)
	// In case of error, do not fail, we want to clean up as much as possible
	if err != nil {
		fmt.Printf("WARNING: cannot restore environment: %s\n", err)
	}

	if c.previousBaseDir != "" {
		err = os.RemoveAll(c.previousBaseDir)
		// In case of error, do not fail, we want to clean up as much as possible
		if err != nil {
			fmt.Printf("WARNING: cannot clean dir %s: %s\n", c.previousBaseDir, err)
		}
	}

	return nil
}

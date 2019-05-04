// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

// Constants used throughout the tests
const (
	validSHASum   = ""
	invalidSHASum = "not a SHA sum"
	validPath     = ""
	invalidPath   = "not an image"
	cacheCustom   = "/tmp/customcachedir"
)

// createTempCache create a valid Singularity cache in a temporary directory to ease testing
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

func getDefaultCacheValues(t *testing.T) (string, string) {
	me, err := test.GetCurrentUser(t)
	if me == nil || err != nil {
		t.Fatal("failed getting the current user")
	}

	expectedDefaultCache := filepath.Join(me.HomeDir, ".singularity", "cache")
	expectedCustomCache := filepath.Join(cacheCustom, "cache")

	return expectedDefaultCache, expectedCustomCache
}

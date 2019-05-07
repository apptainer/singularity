// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package test

import (
	"testing"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

func TestCacheTestInvalidate(t *testing.T) {
	DropPrivilege(t)
	defer ResetPrivilege(t)

	// Valid case first
	c := CacheTestInit(t)
	if c == nil {
		t.Fatal("cannot initialize cache configuration")
	}
	defer CacheTestFinalize(t, c)

	CacheTestInvalidate(t, c)
	// After invalidating the cache, the base directory is not supposed to be
	// a directory
	if fs.IsDir(c.BaseDir) {
		t.Fatal("cache invalidation failed")
	}

	// Error case: we pass an undefined cache configuration
	err := CacheTestInvalidate(t, nil)
	if err == nil {
		t.Fatal("CacheTestInvalidate() succeeded with an undefined cache configuration")
	}
}

func TestCacheTestInit(t *testing.T) {
	DropPrivilege(t)
	defer ResetPrivilege(t)

	c := CacheTestInit(t)
	if c == nil {
		t.Fatal("cannot initialize cache configuration")
	}
	defer CacheTestFinalize(t, c)

	tempCache, err := cache.Create()
	if tempCache == nil || err != nil {
		t.Fatal("cannot create temporary cache")
	}

	// Some basic checks
	if tempCache.BaseDir != c.BaseDir {
		t.Fatal("base directories do not match")
	}

	err = tempCache.Destroy()
	if err != nil {
		t.Fatal("cannot destroy cache")
	}
}

// TestCaseTestFinalize focuses on error cases, the successful path is tested
// in the context of the other tests
func TestCaseTestFinalize(t *testing.T) {
	err := CacheTestFinalize(t, nil)
	if err == nil {
		t.Fatal("CacheTestFinalize() succeeded with an undefined configuration")
	}
}

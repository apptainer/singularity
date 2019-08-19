// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"strconv"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

const (
	validSHASum   = ""
	invalidSHASum = "not a SHA sum"
	validPath     = ""
	invalidPath   = "not an image"
)

// checkIfCacheDisabled is used throughout the unit tests to check whether the
// SINGULARITY_DISABLE_CACHE environment variable is set. If it is set, it will
// skip the current tests since exercising the cache when caching is disabled is
// not supported.
func (c *Handle) checkIfCacheDisabled(t *testing.T) {
	envValue := os.Getenv(DisableEnv)
	if envValue == "" {
		envValue = "0"
	}
	disabled, err := strconv.ParseBool(envValue) // strconv.ParseBool("") raises an error
	if err != nil {
		t.Fatalf("failed to parse the %s environment variable: %s", DisableEnv, err)
	}
	if disabled {
		t.Skip("Caching is disabled")
	}

	// Before running the test we make sure that the test environment
	// did not implicitly disable the cache.
	if c.IsDisabled() {
		writable, _ := fs.IsWritable(c.GetBasedir())
		if !writable {
			t.Skip("cache's base directory is not writable; cache is disabled")
		}
		t.Skip("cache disabled")
	}
}

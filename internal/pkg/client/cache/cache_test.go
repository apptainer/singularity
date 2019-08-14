// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"strconv"
	"testing"
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
func chechIfCacheDisabled(t *testing.T) {
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
}

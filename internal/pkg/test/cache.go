// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package test

import (
	"io/ioutil"
	"os"
	"testing"
)

// SetCacheDir creates a new image cache in the context of testing
// and set the appropriate environment variable to ensure that the
// temporary cache is used by the caller. Note that the context of
// the directory that is passed in will be deleted when calling
// CleanCacheDir().
func SetCacheDir(t *testing.T, basedir string) string {
	if basedir == "" {
		// We create a unique temporary directory for the image cache since Go run
		// tests concurrently and since SINGULARITY_TESTBASEDIR is a static directory.
		dir, err := ioutil.TempDir("", "image_cache-")
		if err != nil {
			t.Fatalf("failed to create temporary directory: %s", err)
		}

		return dir
	}

	return basedir
}

// CleanCacheDir deleted the temporary cache that was created for
// testing purposes.
func CleanCacheDir(t *testing.T, basedir string) {
	// We clean the image cache only if we are not using the
	// default cache; we do not want to clean the developer's
	// default cache.
	if basedir != "" {
		os.RemoveAll(basedir)
	}
}

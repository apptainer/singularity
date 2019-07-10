// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

// MakeDir creates a new temporary image cache directory in the
// directory basedir and returns the path of the new directory. If
// basedir is an empty string, MakeDir uses the default directory
// for temporary files (see os.TempDir).
// It is the caller's responsibility to call cache.InitHdl after calling
// MakeDir() in order to create a valid cache handle.
// It is also the caller's responsibility to remove the directory when no
// longer needed, which can be accomplished by calling DeleteDir.
// It's the caller's responsibility to create basedir before calling it.
func MakeDir(t *testing.T, basedir string) string {
	// We create a unique temporary directory for the image cache since Go run
	// tests concurrently.
	dir, err := fs.MakeTmpDir(basedir, "image_cache-", 0755)
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// DeleteDir deleted the temporary image cache that was created for
// testing purposes. DeleteDir will fail the test if an error occurs
// during its execution.
func DeleteDir(t *testing.T, path string) {
	// Note: if path is empty os.RemoveAll() will fail
	err := os.RemoveAll(path)
	if err != nil {
		t.Fatalf("failed to remove directory %s: %s", path, err)
	}
}

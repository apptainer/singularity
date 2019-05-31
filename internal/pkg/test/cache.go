// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package test

import (
	"io/ioutil"
	"os"
	"syscall"
	"testing"
)

// SetCacheDir creates a new temporary image cache directory in the
// directory basedir and returns the path of the new directory. If
// basedir is an empty string, SetCacheDir uses the default directory
// for temporary files (see os.TempDir). It is the caller's responsibility
// to remove the directory when no longer needed, which can be accomplished
// by calling CleanCacheDir
func SetCacheDir(t *testing.T, basedir string) string {
	// We create a unique temporary directory for the image cache since Go run
	// tests concurrently.
	dir, err := ioutil.TempDir(basedir, "image_cache-")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}

	// Update the access mode to the directory since ioutil.TempDir()
	// creates a new directory with a different access mode than the
	// one we want here.
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)
	err = os.Chmod(dir, 0755)
	if err != nil {
		t.Fatalf("failed to change permission of %s: %s", dir, err)
	}

	return dir
}

// CleanCacheDir deleted the temporary image cache that was created for
// testing purposes. CleanCacheDir will fail the test if an error occurs
// during its execution.
func CleanCacheDir(t *testing.T, path string) {
	// Note: if path is empty os.RemoveAll() will fail
	err := os.RemoveAll(path)
	if err != nil {
		t.Fatalf("failed to remove directory %s: %s", path, err)
	}
}

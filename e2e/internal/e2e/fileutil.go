// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

// WriteTempFile creates and populates a temporary file in the specified
// directory or in os.TempDir if dir is ""
// returns the file name or an error
func WriteTempFile(dir, pattern, content string) (string, error) {
	tmpfile, err := ioutil.TempFile(dir, pattern)
	if err != nil {
		return "", err
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		return "", err
	}

	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	return tmpfile.Name(), nil
}

// MakeCacheDir creates a temporary image cache directory that can then be
// used for the execution of a e2e test.
//
// This function shall not set the environment variable to specify the
// image cache location since it would create thread safety problems.
func makeTempDir(t *testing.T, baseDir string, prefix string, context string) (string, func(t *testing.T)) {
	dir, err := fs.MakeTmpDir(baseDir, prefix, 0755)
	err = errors.Wrapf(err, "creating temporary %s at %s", context, baseDir)
	if err != nil {
		t.Fatalf("failed to create temporary directory: %+v", err)
	}

	return dir, func(t *testing.T) {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("failed to delete temporary directory: %s", err)
		}
	}
}

// MakeCacheDir creates a temporary image cache directory that can then be
// used for the execution of a e2e test.
//
// This function shall not set the environment variable to specify the
// image cache location since it would create thread safety problems.
func MakeCacheDir(t *testing.T, baseDir string) (string, func(t *testing.T)) {
	return makeTempDir(t, baseDir, "e2e-imgcache-", "image cache directory")
}

// MakeSyPGPDir creates a temporary directory that will be used to store the PGP
// keyring for the execution of a e2e test.
//
// This function shall not set the environment variable to specify the
// SYPGP directory since it would create thread safety problems.
func MakeSyPGPDir(t *testing.T, baseDir string) (string, func(t *testing.T)) {
	return makeTempDir(t, baseDir, "e2e-sypgp-", "SyPGP directory")
}

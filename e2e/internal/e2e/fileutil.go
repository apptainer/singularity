// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.
package e2e

import (
	"io/ioutil"
	"os"
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

// MakeTmpDir creates a temporary directory with provided mode
// in os.TempDir if dir is ""
func MakeTmpDir(dir, pattern string, mode os.FileMode) (string, error) {
	name, err := ioutil.TempDir(dir, pattern)
	if err != nil {
		return "", err
	}
	if err := os.Chmod(name, mode); err != nil {
		return "", err
	}
	return name, nil
}

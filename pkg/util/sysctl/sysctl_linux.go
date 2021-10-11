// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sysctl

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const procSys = "/proc/sys"

func convertKey(key string) string {
	return strings.Replace(strings.TrimSpace(key), ".", string(os.PathSeparator), -1)
}

func getPath(key string) (string, error) {
	path := filepath.Join(procSys, convertKey(key))
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}

// Get retrieves and returns sysctl key value
func Get(key string) (string, error) {
	var path string

	path, err := getPath(key)
	if err != nil {
		return "", fmt.Errorf("can't retrieve key %s: %s", key, err)
	}

	value, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("can't retrieve value for key %s: %s", key, err)
	}

	return strings.TrimSuffix(string(value), "\n"), nil
}

// Set sets value for sysctl key value
func Set(key string, value string) error {
	var path string

	path, err := getPath(key)
	if err != nil {
		return fmt.Errorf("can't retrieve key %s: %s", key, err)
	}

	return ioutil.WriteFile(path, []byte(value), 0o000)
}

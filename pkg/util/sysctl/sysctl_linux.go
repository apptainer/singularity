// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sysctl

import (
	"bufio"
	"fmt"
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
	var file *os.File

	path, err := getPath(key)
	if err != nil {
		return "", fmt.Errorf("can't retrieve key %s: %s", key, err)
	}

	file, err = os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return "", fmt.Errorf("can't retrieve value for key %s: %s", key, err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	value, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("can't read value for key %s: %s", key, err)
	}

	return strings.TrimSuffix(value, "\n"), nil
}

// Set sets value for sysctl key value
func Set(key string, value string) error {
	var path string
	var file *os.File

	path, err := getPath(key)
	if err != nil {
		return fmt.Errorf("can't retrieve key %s: %s", key, err)
	}

	file, err = os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("can't set value for key %s: %s", key, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	_, err = writer.WriteString(value)
	if err != nil {
		return fmt.Errorf("can't set value for key %s: %s", key, err)
	}

	return nil
}

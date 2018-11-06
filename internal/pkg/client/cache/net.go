// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"path/filepath"
)

const (
	// NetDir is the directory inside the cache.Dir where net images are cached
	NetDir = "net"
)

// Net returns the directory inside the cache.Dir() where shub images are cached
func Net() string {
	return updateCacheSubdir(NetDir)
}

// NetImage creates a directory inside cache.Dir() with the name of the SHA sum of the image
func NetImage(sum, name string) string {
	updateCacheSubdir(filepath.Join(NetDir, sum))

	return filepath.Join(Net(), sum, name)
}

// NetImageExists returns whether the image with the SHA sum exists in the net cache
func NetImageExists(sum, name string) (bool, error) {
	_, err := os.Stat(NetImage(sum, name))
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

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
	// ShubDir is the directory inside the cache.Dir where shub images are cached
	ShubDir = "shub"
)

// Shub returns the directory inside the cache.Dir() where shub images are cached
func Shub() string {
	return updateCacheSubdir(ShubDir)
}

// ShubImage creates a directory inside cache.Dir() with the name of the SHA sum of the image
func ShubImage(sum, name string) string {
	updateCacheSubdir(filepath.Join(ShubDir, sum))

	return filepath.Join(Shub(), sum, name)
}

// ShubImageExists returns whether the image with the SHA sum exists in the ShubImage cache
func ShubImageExists(sum, name string) (bool, error) {
	_, err := os.Stat(ShubImage(sum, name))
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

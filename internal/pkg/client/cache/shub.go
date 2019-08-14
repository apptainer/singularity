// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
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
func getShubCachePath(c *Handle) (string, error) {
	// This function may act on an cache object that is not fully initialized
	// so it is not a method on a Handle but rather an independent
	// function

	// updateCacheSubdir checks if the cache is valid, no need to check here
	return updateCacheSubdir(c, ShubDir)
}

// ShubImage creates a directory inside cache.Dir() with the name of the SHA sum of the image
func (c *Handle) ShubImage(sum, name string) string {
	if c.disabled {
		return ""
	}

	_, err := updateCacheSubdir(c, filepath.Join(ShubDir, sum))
	if err != nil {
		return ""
	}

	return filepath.Join(c.Shub, sum, name)
}

// ShubImageExists returns whether the image with the SHA sum exists in the ShubImage cache
func (c *Handle) ShubImageExists(sum, name string) (bool, error) {
	if c.disabled {
		return false, nil
	}

	_, err := os.Stat(c.ShubImage(sum, name))
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/oras"
)

const (
	// OrasDir is the directory inside the cache.Dir where oras images are cached
	OrasDir = "oras"
)

// Shub returns the directory inside the cache.Dir() where shub images are cached
func getOrasCachePath(c *Handle) (string, error) {
	// This function may act on a cache object that is not fully initialized
	// so it is not a method on a Handle but rather an independent
	// function

	// updateCacheSubdir checks if the cache is valid, no need to check here
	return updateCacheSubdir(c, OrasDir)
}

// OrasImage creates a directory inside cache.Dir() with the name of the SHA sum of the image
func (c *Handle) OrasImage(sum, name string) string {
	dir, err := updateCacheSubdir(c, filepath.Join(OrasDir, sum))
	if err != nil {
		return ""
	}

	return filepath.Join(dir, name)
}

// OrasImageExists returns whether the image with the SHA sum exists in the OrasImage cache
func (c *Handle) OrasImageExists(sum, name string) (bool, error) {
	imagePath := c.OrasImage(sum, name)
	_, err := os.Stat(imagePath)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	cacheSum, err := oras.ImageHash(imagePath)
	if err != nil {
		return false, err
	}
	if cacheSum != sum {
		return false, ErrBadChecksum
	}

	return true, nil
}

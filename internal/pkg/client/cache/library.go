// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"path/filepath"

	"github.com/sylabs/scs-library-client/client"
)

const (
	// LibraryDir is the directory inside the cache.Dir where library images are cached
	LibraryDir = "library"
)

// Library returns the directory inside the cache.Dir() where library
// images are cached
func getLibraryCachePath(c *Handle) (string, error) {
	// This function may act on an cache object that is not fully
	// initialized so it is not a method on a Handle but
	// rather an independent function.

	return updateCacheSubdir(c, LibraryDir)
}

// LibraryImage creates a directory inside cache.Dir() with the name of the SHA sum of the image.
func (c *Handle) LibraryImage(sum, name string) string {
	if c.disabled {
		return ""
	}

	_, err := updateCacheSubdir(c, filepath.Join(LibraryDir, sum))
	if err != nil {
		return ""
	}

	return filepath.Join(c.Library, sum, name)
}

// LibraryImageExists returns whether the image with the SHA sum exists in the LibraryImage cache.
func (c *Handle) LibraryImageExists(sum, name string) (bool, error) {
	if c.disabled {
		return false, nil
	}

	imagePath := c.LibraryImage(sum, name)
	_, err := os.Stat(imagePath)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	cacheSum, err := client.ImageHash(imagePath)
	if err != nil {
		return false, err
	}
	if cacheSum != sum {
		return false, ErrBadChecksum
	}

	return true, nil
}

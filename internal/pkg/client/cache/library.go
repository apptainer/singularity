// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

const (
	// LibraryDir is the directory inside the cache.Dir where library images are cached
	LibraryDir = "library"
)

// Library returns the directory inside the cache.Dir() where library
// images are cached
func getLibraryCachePath(c *SingularityCache) (string, error) {
	// This function may act on an cache object that is not fully
	// initialized so it is not a method on a SingularityCache but
	// rather an independent function.

	return updateCacheSubdir(c, LibraryDir)
}

// LibraryImage creates a directory inside cache.Dir() with the name
// of the SHA sum of the image
func (c *SingularityCache) LibraryImage(sum, name string) (string, error) {
	if sum == "" || name == "" {
		return "", fmt.Errorf("invalid parameters")
	}

	_, err := updateCacheSubdir(c, filepath.Join(LibraryDir, sum))
	if err != nil {
		return "", fmt.Errorf("failed to update cache sub-directory: %s", err)
	}

	return filepath.Join(c.Library, sum, name), nil
}

// LibraryImageExists returns whether the image with the SHA sum
// exists in the LibraryImage cache.
func (c *SingularityCache) LibraryImageExists(sum, name string) (bool, error) {
	if !c.isValid() {
		return false, fmt.Errorf("invalid cache")
	}

	imagePath, err := c.LibraryImage(sum, name)
	if err != nil {
		return false, err
	}

	exists, err := fs.Exists(imagePath)
	if !exists || err != nil {
		return false, err
	}

	if !checkImageHash(imagePath, sum) {
		return false, fmt.Errorf("invalid image sum: %s", sum)
	}

	return true, nil
}

// cleanLibraryCache deletes the cache's sub-directory used for the library cache.
func (c *SingularityCache) cleanLibraryCache() error {
	if !c.isValid() {
		return fmt.Errorf("invalid cache")
	}

	sylog.Debugf("Removing: %v", c.Library)

	err := os.RemoveAll(c.Library)
	if err != nil {
		return fmt.Errorf("unable to clean library cache: %v", err)
	}

	return nil
}

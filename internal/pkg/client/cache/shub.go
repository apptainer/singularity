// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"fmt"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

const (
	// ShubDir is the directory inside the cache.Dir where shub images are cached
	ShubDir = "shub"
)

// Shub returns the directory inside the cache.Dir() where shub images are cached
func getShubCachePath(c *SingularityCache) (string, error) {
	// This function may act on an cache object that is not fully initialized
	// so it is not a method on a SingularityCache but rather an independent
	// function

	// updateCacheSubdir checks if the cache is valid, no need to check here
	return updateCacheSubdir(c, ShubDir)
}

// ShubImage creates a directory inside cache.Dir() with the name of the SHA sum of the image
func (c *SingularityCache) ShubImage(sum, name string) (string, error) {
	if !c.IsValid() {
		return "", fmt.Errorf("invalid cache")
	}

	// the name and sum cannot be empty strings otherwise we have name collision
	// between images and the cache directory itself
	if sum == "" || name == "" {
		return "", fmt.Errorf("invalid parameters")
	}

	_, err := updateCacheSubdir(c, filepath.Join(ShubDir, sum))
	if err != nil {
		return "", fmt.Errorf("failed to update the cache's sub-directory: %s", err)
	}

	return filepath.Join(c.Shub, sum, name), nil
}

// ShubImageExists returns whether the image with the SHA sum exists in the ShubImage cache
func (c *SingularityCache) ShubImageExists(sum, name string) (bool, error) {
	path, err := c.ShubImage(sum, name)
	if err != nil {
		return false, err
	}

	return fs.Exists(path)
}

// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"fmt"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	client "github.com/sylabs/singularity/pkg/client/library"
)

const (
	// LibraryDir is the directory inside the cache.Dir where library images are cached
	LibraryDir = "library"
)

// Library returns the directory inside the cache.Dir() where library images are cached
func (c *SingularityCache) Library() (string, error) {
	// updateCacheSubdir checks if the cache is valid, no need to check here
	return c.updateCacheSubdir(LibraryDir)
}

// LibraryImage creates a directory inside cache.Dir() with the name of the SHA sum of the image
func (c *SingularityCache) LibraryImage(sum, name string) (string, error) {
	// updateCacheSubdir checks if the cache is valid, no need to check here
	_, err := c.updateCacheSubdir(filepath.Join(LibraryDir, sum))
	if err != nil {
		return "", fmt.Errorf("failed to update cache sub-directory: %s", err)
	}

	path, err := c.Library()
	if err != nil {
		return "", fmt.Errorf("failed to get the cache directory for images")
	}

	return filepath.Join(path, sum, name), nil
}

// LibraryImageExists returns whether the image with the SHA sum exists in the LibraryImage cache
func (c *SingularityCache) LibraryImageExists(sum, name string) (bool, error) {
	if c.IsValid() == false {
		return false, fmt.Errorf("invalid cache")
	}

	imagePath, err := c.LibraryImage(sum, name)
	if err != nil {
		return false, fmt.Errorf("failed to get cache information: %s", err)
	}

	exists, err := fs.Exists(imagePath)
	if exists == false || err != nil {
		return exists, err
	}

	cacheSum, err := client.ImageHash(imagePath)
	if err != nil {
		return false, err
	}
	if cacheSum != sum {
		sylog.Debugf("Cached File Sum(%s) and Expected Sum(%s) does not match", cacheSum, sum)
		return false, nil
	}

	return true, nil
}

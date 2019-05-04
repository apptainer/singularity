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
	// NetDir is the directory inside the cache.Dir where net images are cached
	NetDir = "net"
)

// Net returns the directory inside the cache.Dir() where shub images are cached
func (c *SingularityCache) Net() (string, error) {
	// updateCacheSubdir checks if the cache is valid, no need to check here
	return c.updateCacheSubdir(NetDir)
}

// NetImage creates a directory inside cache.Dir() with the name of the SHA sum of the image
func (c *SingularityCache) NetImage(sum, name string) (string, error) {
	// updateCacheSubdir checks if the cache is valid, no need to check here
	path, err := c.updateCacheSubdir(filepath.Join(NetDir, sum))
	if err != nil {
		return "", fmt.Errorf("failed to update cache's sub-directory")
	}

	return filepath.Join(path, sum, name), nil
}

// NetImageExists returns whether the image with the SHA sum exists in the net cache
func (c *SingularityCache) NetImageExists(sum, name string) (bool, error) {
	if c.IsValid() == false {
		return false, fmt.Errorf("invalid cache")
	}

	path, err := c.NetImage(sum, name)
	if err != nil {
		return false, fmt.Errorf("failed to get image's data: %s", err)
	}

	return fs.Exists(path)
}

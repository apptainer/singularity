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
func getNetCachePath(c *SingularityCache) (string, error) {
	// This function may act on an cache object that is not fully initialized
	// so it is not a method on a SingularityCache but rather an independent
	// function

	return updateCacheSubdir(c, NetDir)
}

// NetImage creates a directory inside cache.Dir() with the name of the SHA sum of the image.
// sum and path must not be empty strings since it would create name collisions.
func (c *SingularityCache) NetImage(sum, name string) (string, error) {
	if !c.IsValid() {
		return "", fmt.Errorf("invalid cache")
	}

	// the name and sum cannot be empty strings otherwise we have name collision
	// between images and the cache directory itself
	if sum == "" || name == "" {
		return "", fmt.Errorf("invalid arguments")
	}

	return filepath.Join(c.Net, sum, name), nil
}

// NetImageExists returns whether the image with the SHA sum exists in the net cache
func (c *SingularityCache) NetImageExists(sum, name string) (bool, error) {
	if !c.IsValid() {
		return false, fmt.Errorf("invalid cache")
	}

	path, err := c.NetImage(sum, name)
	if err != nil {
		return false, fmt.Errorf("failed to get image's data: %s", err)
	}

	return fs.Exists(path)
}

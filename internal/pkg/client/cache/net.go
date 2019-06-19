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
	// NetDir is the directory inside the cache.Dir where net images are cached
	NetDir = "net"
)

/*
// Net returns the directory inside the cache.Dir() where shub images are cached
func Net() string {
	return updateCacheSubdir(NetDir)
}
*/

// Net returns the directory inside the cache.Dir() where shub images are
// cached
func getNetCachePath(c *ImgCache) (string, error) {
	// This function may act on an cache object that is not fully initialized
	// so it is not a method on a ImgCache but rather an independent
	// function

	return updateCacheSubdir(c, NetDir)
}

// NetImage creates a directory inside cache.Dir() with the name of the SHA sum of the image
func (c *ImgCache) NetImage(sum, name string) string {
	_, err := updateCacheSubdir(c, filepath.Join(NetDir, sum))
	if err != nil {
		return ""
	}

	return filepath.Join(c.Net, sum, name)
}

// NetImageExists returns whether the image with the SHA sum exists in the net cache
func (c *ImgCache) NetImageExists(sum, name string) (bool, error) {
	_, err := os.Stat(c.NetImage(sum, name))
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

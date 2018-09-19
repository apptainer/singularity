// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

// Shub returns the directory inside the cache.Dir() where shub images are cached
func Shub() string {
	return updateCacheSubdir(ShubDir)
}

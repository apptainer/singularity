// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/oras"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

const (
	// OrasDir is the directory inside the cache.Dir where oras images are cached
	OrasDir = "oras"
)

// Oras returns the directory inside the cache.Dir() where oras images are cached
func Oras() string {
	return updateCacheSubdir(OrasDir)
}

// OrasImage creates a directory inside cache.Dir() with the name of the SHA sum of the image
func OrasImage(sum, name string) string {
	dir := updateCacheSubdir(filepath.Join(OrasDir, sum))

	return filepath.Join(dir, name)
}

// OrasImageExists returns whether the image with the SHA sum exists in the OrasImage cache
func OrasImageExists(sum, name string) (bool, error) {
	imagePath := OrasImage(sum, name)
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
		sylog.Debugf("Cached File Sum(%s) and Expected Sum(%s) does not match", cacheSum, sum)
		return false, nil
	}

	return true, nil
}

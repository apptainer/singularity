// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	client "github.com/sylabs/singularity/pkg/client/library"
)

const (
	// LibraryDir is the directory inside the cache.Dir where library images are cached
	LibraryDir = "library"
)

// Library returns the directory inside the cache.Dir() where library images are cached
func Library() string {
	return updateCacheSubdir(LibraryDir)
}

// LibraryImage creates a directory inside cache.Dir() with the name of the SHA sum of the image
func LibraryImage(sum, name string) string {
	updateCacheSubdir(filepath.Join(LibraryDir, sum))

	return filepath.Join(Library(), sum, name)
}

// LibraryImageExists returns whether the image with the SHA sum exists in the LibraryImage cache
func LibraryImageExists(sum, name string) (bool, error) {
	imagePath := LibraryImage(sum, name)
	_, err := os.Stat(imagePath)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	cacheSum, err := client.ImageHash(imagePath)
	if err != nil {
		sylog.Debugf("Error getting ImageHash - Deleting Cached File: %s", err)
		os.Remove(imagePath)
		return false, nil
	}
	if cacheSum != sum {
		sylog.Debugf("Cached File Sum(%s) and Expected Sum(%s) does not match - Deleting Cached File", cacheSum, sum)
		os.Remove(imagePath)
		return false, nil
	}

	return true, nil
}

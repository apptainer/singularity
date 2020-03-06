// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

// issue5097 - need to handle an existing directory entry present in the cache
// from older singularity versions.
func (c cacheTests) issue5097(t *testing.T) {
	imgCacheDir, cleanCache := e2e.MakeCacheDir(t, c.env.TestDir)
	defer cleanCache(t)
	c.env.ImgCacheDir = imgCacheDir

	tempDir, imgStoreCleanup := e2e.MakeTempDir(t, "", "", "image store")
	defer imgStoreCleanup(t)
	imagePath := filepath.Join(tempDir, imgName)

	// Pull through the cache - will give us a new style file in the cache
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("pull"),
		e2e.WithArgs([]string{"--force", imagePath, imgURL}...),
		e2e.ExpectExit(0),
	)

	// Replace the cache entry with a directory, containing the image,
	// like in older versions of singularity
	hash, err := client.ImageHash(imagePath)
	if err != nil {
		t.Fatalf("Could not calculate hash of test image: %v", err)
	}
	cachePath := path.Join(imgCacheDir, "cache", "library", hash)
	err = os.Remove(cachePath)
	if err != nil {
		t.Fatalf("Could not remove cached image '%s': %v", cachePath, err)
	}
	err = os.Mkdir(cachePath, 0700)
	if err != nil {
		t.Fatalf("Could not create directory '%s': %v", cachePath, err)
	}
	err = fs.CopyFile(imagePath, path.Join(cachePath, hash), 0700)
	if err != nil {
		t.Fatalf("Could not copy file to directory '%s': %v", cachePath, err)
	}

	// Pull through the cache - it should work as we now remove the directory and
	// re-pull a file into the cache
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("pull"),
		e2e.WithArgs([]string{"--force", imagePath, imgURL}...),
		e2e.ExpectExit(0),
	)

	if !fs.IsFile(cachePath) {
		t.Fatalf("Cache entry '%s' is not a file", cachePath)
	}

}

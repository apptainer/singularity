// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package cache provides support for automatic caching of any image supported by containers/image
package cache

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/pkg/syfs"
)

const (
	// DirEnv specifies the environment variable which can set the directory
	// for image downloads to be cached in
	DirEnv = "SINGULARITY_CACHEDIR"

	// cacheDir specifies the name of the directory relative to the
	// singularity data directory where images are cached in by
	// default.
	// Uses "~/.singularity/cache" which will not clash with any 2.x cache
	// directory.
	cacheDir = "cache"
)

var root string

// Root is the root location where all of singularity caching happens. Library, Shub,
// and oci image formats supported by containers/image repository will be cached inside
//
// Defaults to ${HOME}/.singularity/cache
func Root() string {
	updateCacheRoot()

	return root
}

func updateCacheRoot() {
	if d := os.Getenv(DirEnv); d != "" {
		root = d
	} else {
		root = path.Join(syfs.ConfigDir(), cacheDir)
	}

	if err := initCacheDir(root); err != nil {
		sylog.Fatalf("Unable to initialize caching directory: %v", err)
	}
}

func updateCacheSubdir(subdir string) string {
	updateCacheRoot()

	absdir, err := filepath.Abs(filepath.Join(root, subdir))
	if err != nil {
		sylog.Fatalf("Unable to get abs filepath: %v", err)
	}

	if err := initCacheDir(absdir); err != nil {
		sylog.Fatalf("Unable to initialize caching directory: %v", err)
	}

	sylog.Debugf("Caching directory set to %s", absdir)
	return absdir
}

func initCacheDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		sylog.Debugf("Creating cache directory: %s", dir)
		if err := fs.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("couldn't create cache directory %v: %v", dir, err)
		}
	} else if err != nil {
		return fmt.Errorf("unable to stat %s: %s", dir, err)
	}

	return nil
}

// cleanAllCaches is an utility function that wipes all files in the
// cache directory, will return a error if one occurs
func cleanAllCaches() {
	// TODO: add oras here
	cacheDirs := map[string]string{
		"library": Library(),
		"oci":     OciTemp(),
		"blob":    OciBlob(),
		"shub":    Shub(),
	}

	for name, dir := range cacheDirs {
		if err := os.RemoveAll(dir); err != nil {
			sylog.Verbosef("unable to clean %s cache, directory %s: %v", name, dir, err)
		}
	}
}

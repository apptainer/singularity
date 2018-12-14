// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package cache provides support for automatic caching of any image supported by containers/image
package cache

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

const (
	// DirEnv specifies the environment variable which can set the directory
	// for image downloads to be cached in
	DirEnv = "SINGULARITY_CACHEDIR"

	// RootDefault specifies the directory inside of ${HOME} that images are
	// cached in by default.
	// Uses "~/.singularity/cache" which will not clash with any 2.x cache
	// directory.
	RootDefault = ".singularity/cache"
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

// Clean wipes all files in the cache directory
func Clean() {
	_ = os.RemoveAll(Root())
}

func updateCacheRoot() {
	usr, err := user.Current()
	if err != nil {
		sylog.Fatalf("Couldn't determine user home directory: %v", err)
	}

	if d := os.Getenv(DirEnv); d != "" {
		root = d
	} else {
		root = path.Join(usr.HomeDir, RootDefault)
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

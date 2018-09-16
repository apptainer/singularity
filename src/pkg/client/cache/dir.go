// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package cache provides support for automatic caching of any image supported by containers/image
package cache

import (
	"os"
	"os/user"
	"path"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/fs"
)

const (
	// DirEnv specifies the environment variable which can set the directory
	// for image downloads to be cached in
	DirEnv = "SINGULARITY_CACHEDIR"

	// DirDefault specifies the directory inside of ${HOME} that images are
	// cached in by default.
	// Uses "~/.singularity/cache/oci" which will not clash with any 2.x cache
	// directory.
	DirDefault = ".singularity/cache/oci"
)

// Dir is the location of the directory in which to cache blobs downloaded
// from image formats supported by containers/image
//
// Defaults to ${HOME}/.singularity/cache/oci
var Dir string

func init() {
	Dir = cacheDir()
	initCacheDir()
}

func cacheDir() string {
	usr, err := user.Current()
	if err != nil {
		sylog.Fatalf("Couldn't determine user home directory: %v", err)
	}

	if dir := os.Getenv(DirEnv); dir != "" {
		return path.Join(dir, "oci")
	}
	return path.Join(usr.HomeDir, DirDefault)
}

func initCacheDir() {
	if _, err := os.Stat(Dir); os.IsNotExist(err) {
		sylog.Debugf("Creating oci cache directory: %s", Dir)
		if err := fs.MkdirAll(Dir, 0755); err != nil {
			sylog.Fatalf("Couldn't create oci cache directory: %v", err)
		}
	} else if err != nil {
		sylog.Fatalf("Unable to stat %s: %s", Dir, err)
	}
}

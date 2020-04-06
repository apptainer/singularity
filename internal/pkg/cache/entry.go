// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package cache provides support for caching SIF, OCI, SHUB images and any OCI layers used to build them
package cache

import (
	"fmt"
	"os"

	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/pkg/sylog"
)

// Entry is a structure representing an entry in the cache. An entry is a file under the
// CacheType subdir within the Cache rootDir
type Entry struct {
	// cacheType indicates which subcache / subdir the entry belongs to, e.g. 'library'
	CacheType string
	// exists is true if the entry exists in the cache at path
	Exists bool
	// path is the location of the entry if exists is true, or the location that a new entry
	// will take when it is finalized
	Path string
	// tmpPath is the temporary location that should be used for a new cache entry as it
	// is created
	TmpPath string
}

// Finalize an entry by renaming it to its permanent path atomically
func (e *Entry) Finalize() error {
	// Try to rename the temporary file to its permanent path
	// This is a file, so we won't have an IsExist error since...
	//   If newpath already exists and is not a directory, Rename replaces it.
	//   https://golang.org/pkg/os/#Rename
	err := os.Rename(e.TmpPath, e.Path)
	if err != nil {
		return fmt.Errorf("could not finalize cached file: %v", err)
	}
	return nil
}

// CleanTmp should be defer'd when an Entry is created and will remove any temporary file
func (e *Entry) CleanTmp() {
	// If there is no TmpPath / file there then there is nothing to clean up
	if e.TmpPath == "" || !fs.IsFile(e.TmpPath) {
		return
	}
	err := os.Remove(e.TmpPath)
	if err != nil {
		sylog.Errorf("Could not remove cache temporary file '%s': %v", e.TmpPath, err)
	}
}

// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package cache provides support for caching SIF, OCI, SHUB images and any OCI layers used to build them
package cache

import (
	"fmt"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"io"
	"os"
)

// Entry is a structure representing an entry in the cache. An entry is itself a subdirectory under the
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

// Finalize an entry by renaming it's dir to its permanent path atomically
func (e *Entry) Finalize() error {
	// If the permanent path exists we were beaten to it by another process
	// That's fine!
	_, err := os.Stat(e.Path)
	if err == nil {
		return nil
	}

	// Try to rename the temporary directory to its permanent path
	err = os.Rename(e.TmpPath, e.Path)
	// Success
	if err == nil {
		e.Exists = true
		return nil
	}
	// If the permanent path exists we were beaten to it by another process
	// That's fine!
	if os.IsExist(err) {
		e.Exists = true
		return nil
	}
	// Uh-oh - something else went wrong
	return err
}

// Abort will remove the temporary directory, if possible
func (e *Entry) Abort() error {
	return fs.ForceRemoveAll(e.TmpPath)
}

func (e *Entry) CopyTo(dest string) error {
	if ! e.Exists{
		return fmt.Errorf("Cannot copy a cache entry that does not exist / is not finalized")
	}

	// Perms are 755 *prior* to umask in order to allow image to be
	// executed with its leading shebang like a script
	destFile, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("while opening destination file: %s", err)
	}
	defer destFile.Close()

	srcFile, err := os.Open(e.Path)
	if err != nil {
		return fmt.Errorf("while opening cached image: %v", err)
	}
	defer srcFile.Close()

	// Copy SIF from cache
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("while copying image from cache: %v", err)
	}

	return nil

}



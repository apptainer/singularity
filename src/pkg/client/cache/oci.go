// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"path/filepath"
)

const (
	// OciBlobDir is the directory inside cache.Dir() where oci images are cached
	OciBlobDir = "oci"
	// OciTempDir is the directory inside cache.Dir() where splatted out oci images live
	OciTempDir = "oci-tmp"
)

// OciBlob returns the directory inside cache.Dir() where oci blobs are cached
func OciBlob() string {
	return updateCacheSubdir(OciBlobDir)
}

// OciTemp returns the directory inside cache.Dir() where splatted out oci images live
func OciTemp() string {
	return updateCacheSubdir(OciTempDir)
}

// OciTempFile returns the full path to a file within the OciTemp() cache directory
func OciTempFile(f string) string {
	return filepath.Join(OciTemp(), f)
}

// OciTempExists returns whether the image with the given sha sum exists in the OciTemp() cache
func OciTempExists(sum string) (bool, error) {
	_, err := os.Stat(OciTempFile(sum))
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

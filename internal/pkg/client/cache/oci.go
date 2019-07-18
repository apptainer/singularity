// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
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
func getOciBlobCachePath(c *Handle) (string, error) {
	// This function may act on an cache object that is not fully initialized
	// so it is not a method on a Handle but rather an independent
	// function

	return updateCacheSubdir(c, OciBlobDir)
}

// OciTemp returns the directory inside cache.Dir() where splatted out oci
// images live
func getOciTempCachePath(c *Handle) (string, error) {
	// This function may act on an cache object that is not fully initialized
	// so it is not a method on a Handle but rather an independent
	// function

	return updateCacheSubdir(c, OciTempDir)
}

// OciTempImage creates a OciTempDir/sum directory and returns the abs path of the image
func (c *Handle) OciTempImage(sum, name string) string {
	_, err := updateCacheSubdir(c, filepath.Join(OciTempDir, sum))
	if err != nil {
		return ""
	}

	return filepath.Join(c.OciTemp, sum, name)
}

// OciTempExists returns whether the image with the given sha sum exists in the OciTemp() cache
func (c *Handle) OciTempExists(sum, name string) (bool, error) {
	_, err := os.Stat(c.OciTempImage(sum, name))
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

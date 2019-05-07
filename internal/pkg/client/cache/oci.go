// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"fmt"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

const (
	// OciBlobDir is the directory inside cache.Dir() where oci images are cached
	OciBlobDir = "oci"
	// OciTempDir is the directory inside cache.Dir() where splatted out oci images live
	OciTempDir = "oci-tmp"
)

// OciBlob returns the directory inside cache.Dir() where oci blobs are cached
func getOciBlobCachePath(c *SingularityCache) (string, error) {
	// This function may act on an cache object that is not fully initialized
	// so it is not a method on a SingularityCache but rather an independent
	// function

	return updateCacheSubdir(c, OciBlobDir)
}

// OciTemp returns the directory inside cache.Dir() where splatted out oci images live
func getOciTempCachePath(c *SingularityCache) (string, error) {
	// This function may act on an cache object that is not fully initialized
	// so it is not a method on a SingularityCache but rather an independent
	// function

	return updateCacheSubdir(c, OciTempDir)
}

// OciTempImage creates a OciTempDir/sum directory and returns the abs path of the image
func (c *SingularityCache) OciTempImage(sum, name string) (string, error) {
	if !c.IsValid() {
		return "", fmt.Errorf("invalid cache")
	}

	// the name and sum cannot be empty strings otherwise we have name collision
	// between images and the cache directory itself
	if sum == "" || name == "" {
		return "", fmt.Errorf("invalid paramters")
	}

	// updateCacheSubdir checks whether the cache is valid, no need to do it here
	_, err := updateCacheSubdir(c, filepath.Join(OciTempDir, sum))
	if err != nil {
		return "", fmt.Errorf("failed to update the cache's sub-directory: %s", err)
	}

	return filepath.Join(c.OciTemp, sum, name), nil
}

// OciTempExists returns whether the image with the given sha sum exists in the OciTemp() cache
func (c *SingularityCache) OciTempExists(sum, name string) (bool, error) {
	if !c.IsValid() {
		return false, fmt.Errorf("invalid cache")
	}
	path, err := c.OciTempImage(sum, name)
	if err != nil {
		return false, fmt.Errorf("failed to get OCI cache information: %s", err)
	}

	return fs.Exists(path)
}

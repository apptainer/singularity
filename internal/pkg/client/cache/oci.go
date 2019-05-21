// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/sylog"
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

// OciTemp returns the directory inside cache.Dir() where splatted out oci
// images live
func getOciTempCachePath(c *SingularityCache) (string, error) {
	// This function may act on an cache object that is not fully initialized
	// so it is not a method on a SingularityCache but rather an independent
	// function

	return updateCacheSubdir(c, OciTempDir)
}

// OciTempImage creates a OciTempDir/sum directory and returns the abs path of the image
func (c *SingularityCache) OciTempImage(sum, name string) (string, error) {
	if !c.isValid() {
		return "", fmt.Errorf("invalid cache")
	}

	// the name and sum cannot be empty strings otherwise we have name
	// collision between images and the cache directory itself
	if sum == "" || name == "" {
		return "", fmt.Errorf("invalid paramters")
	}

	_, err := updateCacheSubdir(c, filepath.Join(OciTempDir, sum))
	if err != nil {
		return "", fmt.Errorf("failed to update the cache's sub-directory: %s", err)
	}

	return filepath.Join(c.OciTemp, sum, name), nil
}

// OciTempExists returns whether the image with the given sha sum exists in
// the OciTemp cache.
func (c *SingularityCache) OciTempExists(sum, name string) (bool, error) {
	if !c.isValid() {
		return false, fmt.Errorf("invalid cache")
	}

	path, err := c.OciTempImage(sum, name)
	if err != nil {
		return false, fmt.Errorf("failed to get OCI cache information: %s", err)
	}

	// Exists() does not return an error if the image does not exist. In such
	// a case, we return an empty string
	exists, err := fs.Exists(path)
	if !exists || err != nil {
		return false, err
	}

	if !checkImageHash(path, sum) {
		return false, fmt.Errorf("invalid image sum: %s", sum)
	}

	return true, nil
}

// cleanOciCache deletes the content of the OCI cache sub-directory of a
// given Singularity cache.
func (c *SingularityCache) cleanOciCache() error {
	if !c.isValid() {
		return fmt.Errorf("invalid cache")
	}

	sylog.Debugf("Removing: %v", c.OciTemp)

	err := os.RemoveAll(c.OciTemp)
	if err != nil {
		return fmt.Errorf("unable to clean oci-tmp cache: %v", err)
	}

	return nil
}

// cleanBlobCache delates the content of the OCI blob cache sub-directory
// of a given Singularity cache.
func (c *SingularityCache) cleanBlobCache() error {
	if !c.isValid() {
		return fmt.Errorf("invalid cache")
	}

	sylog.Debugf("Removing: %v", c.OciBlob)

	err := os.RemoveAll(c.OciBlob)
	if err != nil {
		return fmt.Errorf("unable to clean oci-blob cache: %v", err)
	}

	return nil
}

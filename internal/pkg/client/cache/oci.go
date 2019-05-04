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
func (c *SingularityCache) OciBlob() (string, error) {
	// updateCacheSubdir() checks whether the cache is valid, no need to do it here
	return c.updateCacheSubdir(OciBlobDir)
}

// OciTemp returns the directory inside cache.Dir() where splatted out oci images live
func (c *SingularityCache) OciTemp() (string, error) {
	// updateCacheSubdir checks whether the cache is valid, no need to do it here
	return c.updateCacheSubdir(OciTempDir)
}

// OciTempImage creates a OciTempDir/sum directory and returns the abs path of the image
func (c *SingularityCache) OciTempImage(sum, name string) (string, error) {
	// updateCacheSubdir checks whether the cache is valid, no need to do it here
	_, err := c.updateCacheSubdir(filepath.Join(OciTempDir, sum))
	if err != nil {
		return "", fmt.Errorf("failed to update the cache's sub-directory: %s", err)
	}

	path, err := c.OciTemp()
	if err != nil {
		return "", fmt.Errorf("failed to get OCI cache information: %s", err)
	}

	return filepath.Join(path, sum, name), nil
}

// OciTempExists returns whether the image with the given sha sum exists in the OciTemp() cache
func (c *SingularityCache) OciTempExists(sum, name string) (bool, error) {
	if c.IsValid() == false {
		return false, fmt.Errorf("invalid cache")
	}
	path, err := c.OciTempImage(sum, name)
	if err != nil {
		return false, fmt.Errorf("failed to get OCI cache information: %s", err)
	}

	return fs.Exists(path)
}

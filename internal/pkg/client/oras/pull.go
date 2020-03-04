// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oras

import (
	"context"
	"fmt"
	"io/ioutil"

	ocitypes "github.com/containers/image/v5/types"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

// Pull will download the image specified by the provided oci reference and store
// it at the location specified by file, it will use credentials if supplied
func Pull(ctx context.Context, imgCache *cache.Handle, pullFrom string, tmpDir string, ociAuth *ocitypes.DockerAuthConfig) (imagePath string, err error) {
	hash, err := ImageSHA(ctx, pullFrom, ociAuth)
	if err != nil {
		return "", fmt.Errorf("failed to get checksum for %s: %s", pullFrom, err)
	}

	if imgCache.IsDisabled() {
		imagePath, err = ioutil.TempDir(tmpDir, "sbuild-tmp-cache-")
		if err != nil {
			return "", fmt.Errorf("unable to create tmp file: %v", err)
		}
		// Dont use cached image
		if err := DownloadImage(imagePath, pullFrom, ociAuth); err != nil {
			return "", fmt.Errorf("unable to Download Image: %v", err)
		}
	} else {
		cacheEntry, err := imgCache.GetEntry(cache.OrasCacheType, hash)
		if err != nil {
			return "", fmt.Errorf("unable to check if %v exists in cache: %v", hash, err)
		}
		if !cacheEntry.Exists {
			sylog.Infof("Downloading oras image")

			if err := DownloadImage(cacheEntry.TmpPath, pullFrom, ociAuth); err != nil {
				return "", fmt.Errorf("unable to Download Image: %v", err)
			}
			if cacheFileHash, err := ImageHash(cacheEntry.TmpPath); err != nil {
				return "", fmt.Errorf("error getting ImageHash: %v", err)
			} else if cacheFileHash != hash {
				_ = cacheEntry.Abort()
				return "", fmt.Errorf("cached file hash(%s) and expected hash(%s) does not match", cacheFileHash, hash)
			}

			err = cacheEntry.Finalize()
			if err != nil {
				return "", err
			}

		} else {
			sylog.Infof("Using cached SIF image")
		}
		imagePath = cacheEntry.Path
	}

	return imagePath, nil
}

// PullToFile will build a SIF image from the specified oci URI and place it at the specified dest
func PullToFile(ctx context.Context, imgCache *cache.Handle, pullTo, pullFrom, tmpDir string, ociAuth *ocitypes.DockerAuthConfig) (sifFile string, err error) {

	src, err := Pull(ctx, imgCache, pullFrom, tmpDir, ociAuth)
	if err != nil {
		return "", fmt.Errorf("error fetching image to cache: %v", err)
	}

	err = fs.CopyFile(src, pullTo, 0755)
	if err != nil {
		return "", fmt.Errorf("error fetching image to cache: %v", err)
	}

	return pullTo, nil
}

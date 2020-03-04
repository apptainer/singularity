// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"context"
	"fmt"
	"io/ioutil"

	ocitypes "github.com/containers/image/v5/types"
	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/build/oci"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

// Pull will build a SIF image from the specified oci URI and place it in the cache
// or at a temporary location within tmpDir if the cache is disabled.
func Pull(ctx context.Context, imgCache *cache.Handle, pullFrom, tmpDir string, ociAuth *ocitypes.DockerAuthConfig, noHTTPS, noCleanUp bool) (imagePath string, err error) {
	sysCtx := &ocitypes.SystemContext{
		OCIInsecureSkipTLSVerify:    noHTTPS,
		DockerInsecureSkipTLSVerify: ocitypes.NewOptionalBool(noHTTPS),
		DockerAuthConfig:            ociAuth,
	}

	hash, err := oci.ImageSHA(ctx, pullFrom, sysCtx)
	if err != nil {
		return "", fmt.Errorf("failed to get checksum for %s: %s", pullFrom, err)
	}

	sylog.Infof("Converting OCI blobs to SIF format")

	if imgCache.IsDisabled() {
		imagePath, err := ioutil.TempDir(tmpDir, "sbuild-tmp-cache-")
		if err != nil {
			return "", fmt.Errorf("unable to create tmp file: %v", err)
		}
		if err := build.ConvertOciToSIF(ctx, imgCache, pullFrom, imagePath, tmpDir, noHTTPS, noCleanUp, ociAuth); err != nil {
			return "", fmt.Errorf("while building SIF from layers: %v", err)
		}
	} else {

		cacheEntry, err := imgCache.GetEntry(cache.OciTempCacheType, hash)
		if err != nil {
			return "", fmt.Errorf("unable to check if %v exists in cache: %v", hash, err)
		}
		if !cacheEntry.Exists {
			sylog.Infof("Converting OCI blobs to SIF format")

			if err := build.ConvertOciToSIF(ctx, imgCache, pullFrom, cacheEntry.TmpPath, tmpDir, noHTTPS, noCleanUp, ociAuth); err != nil {
				return "", fmt.Errorf("while building SIF from layers: %v", err)
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
func PullToFile(ctx context.Context, imgCache *cache.Handle, pullTo, pullFrom, tmpDir string, ociAuth *ocitypes.DockerAuthConfig, noHTTPS, noCleanUp bool) (sifFile string, err error) {

	src, err := Pull(ctx, imgCache, pullFrom, tmpDir, ociAuth, noHTTPS, noCleanUp)
	if err != nil {
		return "", fmt.Errorf("error fetching image to cache: %v", err)
	}

	err = fs.CopyFile(src, pullTo, 0755)
	if err != nil {
		return "", fmt.Errorf("error fetching image to cache: %v", err)
	}

	return pullTo, nil
}

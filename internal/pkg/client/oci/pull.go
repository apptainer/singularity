// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
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
	"github.com/sylabs/singularity/internal/pkg/cache"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/pkg/syfs"
	"github.com/sylabs/singularity/pkg/sylog"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

// pull will build a SIF image into the cache if directTo="", or a specific file if directTo is set.
func pull(ctx context.Context, imgCache *cache.Handle, directTo, pullFrom, tmpDir string, ociAuth *ocitypes.DockerAuthConfig, noHTTPS, noCleanUp bool) (imagePath string, err error) {
	// DockerInsecureSkipTLSVerify is set only if --nohttps is specified to honor
	// configuration from /etc/containers/registries.conf because DockerInsecureSkipTLSVerify
	// can have three possible values true/false and undefined, so we left it as undefined instead
	// of forcing it to false in order to delegate decision to /etc/containers/registries.conf:
	// https://github.com/sylabs/singularity/issues/5172
	sysCtx := &ocitypes.SystemContext{
		OCIInsecureSkipTLSVerify: noHTTPS,
		DockerAuthConfig:         ociAuth,
		AuthFilePath:             syfs.DockerConf(),
		DockerRegistryUserAgent:  useragent.Value(),
	}
	if noHTTPS {
		sysCtx.DockerInsecureSkipTLSVerify = ocitypes.NewOptionalBool(true)
	}

	hash, err := oci.ImageSHA(ctx, pullFrom, sysCtx)
	if err != nil {
		return "", fmt.Errorf("failed to get checksum for %s: %s", pullFrom, err)
	}

	if directTo != "" {
		sylog.Infof("Converting OCI blobs to SIF format")
		if err := build.ConvertOciToSIF(ctx, imgCache, pullFrom, directTo, tmpDir, noHTTPS, noCleanUp, ociAuth); err != nil {
			return "", fmt.Errorf("while building SIF from layers: %v", err)
		}
		imagePath = directTo
	} else {

		cacheEntry, err := imgCache.GetEntry(cache.OciTempCacheType, hash)
		if err != nil {
			return "", fmt.Errorf("unable to check if %v exists in cache: %v", hash, err)
		}
		defer cacheEntry.CleanTmp()
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

// Pull will build a SIF image to the cache or direct to a temporary file if cache is disabled
func Pull(ctx context.Context, imgCache *cache.Handle, pullFrom, tmpDir string, ociAuth *ocitypes.DockerAuthConfig, noHTTPS, noCleanUp bool) (imagePath string, err error) {

	directTo := ""

	if imgCache.IsDisabled() {
		file, err := ioutil.TempFile(tmpDir, "sbuild-tmp-cache-")
		if err != nil {
			return "", fmt.Errorf("unable to create tmp file: %v", err)
		}
		directTo = file.Name()
		sylog.Infof("Downloading library image to tmp cache: %s", directTo)
	}

	return pull(ctx, imgCache, directTo, pullFrom, tmpDir, ociAuth, noHTTPS, noCleanUp)
}

// PullToFile will build a SIF image from the specified oci URI and place it at the specified dest
func PullToFile(ctx context.Context, imgCache *cache.Handle, pullTo, pullFrom, tmpDir string, ociAuth *ocitypes.DockerAuthConfig, noHTTPS, noCleanUp bool) (imagePath string, err error) {

	directTo := ""
	if imgCache.IsDisabled() {
		directTo = pullTo
		sylog.Debugf("Cache disabled, pulling directly to: %s", directTo)
	}

	src, err := Pull(ctx, imgCache, pullFrom, tmpDir, ociAuth, noHTTPS, noCleanUp)
	if err != nil {
		return "", fmt.Errorf("error fetching image to cache: %v", err)
	}

	if directTo == "" {
		// mode is before umask if pullTo doesn't exist
		err = fs.CopyFileAtomic(src, pullTo, 0777)
		if err != nil {
			return "", fmt.Errorf("error copying image out of cache: %v", err)
		}
	}

	return pullTo, nil
}

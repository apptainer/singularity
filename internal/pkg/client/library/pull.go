// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2020-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package library

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/hpcng/singularity/internal/app/singularity"
	"github.com/hpcng/singularity/internal/pkg/cache"
	"github.com/hpcng/singularity/internal/pkg/client"
	"github.com/hpcng/singularity/internal/pkg/util/fs"
	"github.com/hpcng/singularity/pkg/sylog"
	keyclient "github.com/sylabs/scs-key-client/client"
	libclient "github.com/sylabs/scs-library-client/client"
)

// ErrLibraryPullUnsigned indicates that the interactive portion of the pull was aborted.
var ErrLibraryPullUnsigned = errors.New("failed to verify container")

// pull will pull a library image into the cache if directTo="", or a specific file if directTo is set.
func pull(ctx context.Context, imgCache *cache.Handle, directTo string, imageRef *libclient.Ref, arch string, libraryConfig *libclient.Config) (imagePath string, err error) {
	c, err := libclient.NewClient(libraryConfig)
	if err != nil {
		return "", fmt.Errorf("unable to initialize client library: %v", err)
	}

	ref := fmt.Sprintf("%s:%s", imageRef.Path, imageRef.Tags[0])

	libraryImage, err := c.GetImage(ctx, arch, ref)
	if err == libclient.ErrNotFound {
		return "", fmt.Errorf("image does not exist in the library: %s (%s)", ref, arch)
	}
	if err != nil {
		return "", err
	}

	if directTo != "" {
		sylog.Infof("Downloading library image")
		if err = DownloadImage(ctx, c, directTo, arch, imageRef, &client.DownloadProgressBar{}); err != nil {
			return "", fmt.Errorf("unable to download image: %v", err)
		}
		imagePath = directTo

	} else {
		cacheEntry, err := imgCache.GetEntry(cache.LibraryCacheType, libraryImage.Hash)
		if err != nil {
			return "", fmt.Errorf("unable to check if %v exists in cache: %v", libraryImage.Hash, err)
		}
		defer cacheEntry.CleanTmp()
		if !cacheEntry.Exists {
			sylog.Infof("Downloading library image")

			if err := DownloadImage(ctx, c, cacheEntry.TmpPath, arch, imageRef, &client.DownloadProgressBar{}); err != nil {
				return "", fmt.Errorf("unable to download image: %v", err)
			}

			if cacheFileHash, err := libclient.ImageHash(cacheEntry.TmpPath); err != nil {
				return "", fmt.Errorf("error getting image hash: %v", err)
			} else if cacheFileHash != libraryImage.Hash {
				return "", fmt.Errorf("cached file hash(%s) and expected hash(%s) does not match", cacheFileHash, libraryImage.Hash)
			}

			err = cacheEntry.Finalize()
			if err != nil {
				return "", err
			}
		} else {
			sylog.Infof("Using cached image")
		}
		imagePath = cacheEntry.Path
	}

	return imagePath, nil
}

// Pull will pull a library image to the cache or direct to a temporary file if cache is disabled
func Pull(ctx context.Context, imgCache *cache.Handle, pullFrom *libclient.Ref, arch string, tmpDir string, libraryConfig *libclient.Config) (imagePath string, err error) {
	directTo := ""

	if imgCache.IsDisabled() {
		file, err := ioutil.TempFile(tmpDir, "sbuild-tmp-cache-")
		if err != nil {
			return "", fmt.Errorf("unable to create tmp file: %v", err)
		}
		directTo = file.Name()
		sylog.Infof("Downloading library image to tmp cache: %s", directTo)
	}

	return pull(ctx, imgCache, directTo, pullFrom, arch, libraryConfig)
}

// PullToFile will pull a library image to the specified location, through the cache, or directly if cache is disabled
func PullToFile(ctx context.Context, imgCache *cache.Handle, pullTo string, pullFrom *libclient.Ref, arch string, tmpDir string, libraryConfig *libclient.Config, co []keyclient.Option) (imagePath string, err error) {
	directTo := ""
	if imgCache.IsDisabled() {
		directTo = pullTo
		sylog.Debugf("Cache disabled, pulling directly to: %s", directTo)
	}

	src, err := pull(ctx, imgCache, directTo, pullFrom, arch, libraryConfig)
	if err != nil {
		return "", fmt.Errorf("error fetching image: %v", err)
	}

	if directTo == "" {
		// mode is before umask if pullTo doesn't exist
		err = fs.CopyFileAtomic(src, pullTo, 0o777)
		if err != nil {
			return "", fmt.Errorf("error copying image out of cache: %v", err)
		}
	}

	if err := singularity.Verify(ctx, pullTo, singularity.OptVerifyUseKeyServer(co...)); err != nil {
		sylog.Warningf("%v", err)
		return pullTo, ErrLibraryPullUnsigned
	}

	return pullTo, nil
}

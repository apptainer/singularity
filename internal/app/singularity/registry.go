// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"context"
	"fmt"
	scs "github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/library"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/signing"
)

var errNotInCache = fmt.Errorf("image was not found in cache")

// Library is a Registry implementation for Sylabs Cloud Library.
type Library struct {
	keystoreURI string

	client *scs.Client
	cache  *cache.Handle
}

// NewLibrary initializes and returns new Library ready to  be used.
func NewLibrary(scsConfig *scs.Config, cache *cache.Handle, keystoreURI string) (*Library, error) {
	libraryClient, err := scs.NewClient(scsConfig)
	if err != nil {
		return nil, fmt.Errorf("could not initialize library client: %v", err)
	}

	return &Library{
		keystoreURI: keystoreURI,
		client:      libraryClient,
		cache:       cache,
	}, nil
}

// Pull will download the image from the library.
// After downloading, the image will be checked for a valid signature.
func (l *Library) Pull(ctx context.Context, from, to, arch string) error {
	// strip leading "library://" and append default tag, as necessary
	libraryPath := library.NormalizeLibraryRef(from)

	// check if image exists in library
	imageMeta, err := l.client.GetImage(ctx, arch, libraryPath)
	if err == scs.ErrNotFound {
		return fmt.Errorf("image %s (%s) does not exist in the library", libraryPath, arch)
	}
	if err != nil {
		return fmt.Errorf("could not get image info: %v", err)
	}

	if l.cache.IsDisabled() {
		if err := l.pullAndVerify(ctx, imageMeta, libraryPath, to, arch); err != nil {
			return fmt.Errorf("unable to download image: %s", err)
		}
	} else {
		cacheEntry, err := l.cache.GetEntry(cache.LibraryCacheType, imageMeta.Hash)
		if err != nil {
			return fmt.Errorf("unable to check if %v exists in cache: %v", imageMeta.Hash, err)
		}
		if !cacheEntry.Exists {
			sylog.Infof("Downloading library image")
			go interruptCleanup(func() {
				_ = cacheEntry.Abort()
			})

			if err := l.pullAndVerify(ctx, imageMeta, libraryPath, cacheEntry.TmpPath, arch); err != nil {
				return fmt.Errorf("unable to download image: %s", err)
			}

			err = cacheEntry.Finalize()
			if err != nil {
				return err
			}

		} else {
			sylog.Infof("Using cached image")

		}

		cacheEntry.CopyTo(to)
	}

	_, err = signing.IsSigned(ctx, to, l.keystoreURI, l.client.AuthToken)
	if err != nil {
		sylog.Warningf("%v", err)
		return ErrLibraryPullUnsigned
	}

	sylog.Infof("Download complete: %s\n", to)
	return nil
}

// pullAndVerify downloads library image and verifies it by comparing checksum
// in imgMeta with actual checksum of the downloaded file. The resulting image
// will be saved to the location provided.
func (l *Library) pullAndVerify(ctx context.Context, imgMeta *scs.Image, from, to, arch string) error {
	sylog.Infof("Downloading library image")

	err := library.DownloadImage(ctx, l.client, to, arch, from, printProgress)
	if err != nil {
		return fmt.Errorf("unable to download image: %v", err)
	}

	fileHash, err := scs.ImageHash(to)
	if err != nil {
		return fmt.Errorf("error getting image hash: %v", err)
	}
	if fileHash != imgMeta.Hash {
		return fmt.Errorf("file hash(%s) and expected hash(%s) does not match", fileHash, imgMeta.Hash)
	}
	return nil
}


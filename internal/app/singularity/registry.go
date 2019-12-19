// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	scs "github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/library"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
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

	dst, tmpName, err := func() (string, string, error) {
		dst := to
		if !l.cache.IsDisabled() {
			imageName := uri.GetName("library://" + libraryPath)
			dst = l.cache.LibraryImage(imageMeta.Hash, imageName)

			// here we can check if the file is already in
			// the cache
			if _, err := os.Stat(dst); err == nil {
				// we have the file in the cache, return
				// the same name for the final
				// destination and the temporary
				// location to signal that no rename is
				// necessary
				return dst, dst, nil
			}
		}

		tmpHandle, err := ioutil.TempFile(filepath.Dir(dst), filepath.Base(dst)+".")
		if err != nil {
			return "", "", fmt.Errorf("unable to create temporary image: %w", err)
		}
		tmpHandle.Close()

		tmpName := tmpHandle.Name()

		// This is racy
		if err := os.Remove(tmpName); err != nil {
			return "", "", fmt.Errorf("unable to remove temporary file %s: %w", tmpName, err)
		}

		go interruptCleanup(tmpName)

		return dst, tmpName, nil
	}()

	if err != nil {
		return fmt.Errorf("unable to obtain intermediate location for %s: %w", to, err)
	}

	// if the tmpName is the same as dst, that means we are
	// looking at a file present in the cache, so we can skip
	// downloading it
	if tmpName != dst {
		sylog.Debugf("Downloading to %s for intermediate destination %s and final destination %s", tmpName, dst, to)
		// Download the image, either to the cache, or to the
		// temporary location
		if err := l.pullAndVerify(ctx, imageMeta, libraryPath, tmpName, arch); err != nil {
			return fmt.Errorf("unable to download image: %s", err)
		}

		sylog.Debugf("Renaming temporary file %s to %s", tmpName, dst)
		os.Rename(tmpName, dst)
	}

	// now we either have the image in the correct location (dst ==
	// to) or we have the image in the cache (dst != to). In the
	// later case we need to copy from the cache to the final
	// destination.
	if dst != to {
		os.Remove(to)
		sylog.Debugf("Copying %s to %s", dst, to)
		if err := fs.CopyFile(dst, to, 0755); err != nil {
			return fmt.Errorf("cannot copy cache element %s to final destination %s: %w", dst, to, err)
		}
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
	go interruptCleanup(to)

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

// copyFromCache checks whether an image with the given name and checksum exists in the cache.
// If so, the image will be copied to the location provided.
func (l *Library) copyFromCache(hash, name, to string) error {
	exists, err := l.cache.LibraryImageExists(hash, name)
	if err == cache.ErrBadChecksum {
		sylog.Warningf("Removing cached image: %s: cache could be corrupted", name)
		err := os.Remove(l.cache.LibraryImage(hash, name))
		if err != nil {
			return fmt.Errorf("unable to remove corrupted image from cache: %s", err)
		}
	} else if err != nil {
		return fmt.Errorf("unable to check if %s exists: %v", name, err)
	}

	if !exists {
		return errNotInCache
	}

	// Remove the 'to' image if exists, (before copying from cache).
	err = os.Remove(to)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("unable to remote old image to overide: %s", err)
	}

	from := l.cache.LibraryImage(hash, name)
	// Perms are 755 *prior* to umask in order to allow image to be
	// executed with its leading shebang like a script
	err = fs.CopyFile(from, to, 0755)
	if err != nil {
		return fmt.Errorf("while copying image from cache: %v", err)
	}
	return nil
}

// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"context"
	"fmt"
	"io/ioutil"
	"runtime"

	"github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/internal/pkg/library"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/signing"
)

// LibraryConveyorPacker only needs to hold a packer to pack the image it pulls
// as well as extra information about the library it's pulling from
type LibraryConveyorPacker struct {
	LocalPacker
}

// Get downloads container from Sylabs Cloud Library.
func (cp *LibraryConveyorPacker) Get(ctx context.Context, b *types.Bundle) (err error) {
	sylog.Debugf("Getting container from Library")

	if b.Opts.ImgCache == nil {
		return fmt.Errorf("invalid image cache")
	}

	// check for custom library from definition
	customLib, ok := b.Recipe.Header["library"]
	if ok {
		sylog.Debugf("Using custom library: %v", customLib)
		b.Opts.LibraryURL = customLib
	}

	sylog.Debugf("LibraryURL: %v", b.Opts.LibraryURL)
	sylog.Debugf("LibraryRef: %v", b.Recipe.Header["from"])

	libraryClient, err := client.NewClient(&client.Config{
		BaseURL:   b.Opts.LibraryURL,
		AuthToken: b.Opts.LibraryAuthToken,
	})
	if err != nil {
		return err
	}

	imageRef := library.NormalizeLibraryRef(b.Recipe.Header["from"])

	libraryImage, err := libraryClient.GetImage(ctx, runtime.GOARCH, imageRef)
	if err == client.ErrNotFound {
		return fmt.Errorf("image does not exist in the library: %s (%s)", imageRef, runtime.GOARCH)
	}
	if err != nil {
		return fmt.Errorf("while getting image info: %v", err)
	}

	imageName := uri.GetName("library://" + imageRef)

	var imagePath string
	if b.Opts.NoCache {
		file, err := ioutil.TempFile(b.TmpDir, "sbuild-tmp-cache-")
		if err != nil {
			return fmt.Errorf("unable to create tmp file: %v", err)
		}

		imagePath = file.Name()

		sylog.Infof("Downloading library image to tmp cache: %s", imagePath)

		if err = library.DownloadImageNoProgress(ctx, libraryClient, imagePath, runtime.GOARCH, imageRef); err != nil {
			return fmt.Errorf("unable to download image: %v", err)
		}
	} else {
		imagePath = b.Opts.ImgCache.LibraryImage(libraryImage.Hash, imageName)

		if exists, err := b.Opts.ImgCache.LibraryImageExists(libraryImage.Hash, imageName); err != nil {
			return fmt.Errorf("unable to check if %v exists: %v", imagePath, err)
		} else if !exists {
			sylog.Infof("Downloading library image")

			if err := library.DownloadImageNoProgress(ctx, libraryClient, imagePath, runtime.GOARCH, imageRef); err != nil {
				return fmt.Errorf("unable to download image: %v", err)
			}

			if cacheFileHash, err := client.ImageHash(imagePath); err != nil {
				return fmt.Errorf("error getting image hash: %v", err)
			} else if cacheFileHash != libraryImage.Hash {
				return fmt.Errorf("cached file hash(%s) and expected Hash(%s) does not match", cacheFileHash, libraryImage.Hash)
			}
		}
	}

	_, err = signing.IsSigned(context.Background(), imagePath, b.Opts.KeyStoreURL, 0, false, b.Opts.LibraryAuthToken)
	if err != nil {
		sylog.Warningf("Unable to verify library://%s  ", imageRef)
		sylog.Warningf("Skipping container verification")
	}

	// insert base metadata before unpacking fs
	if err = makeBaseEnv(b.RootfsPath); err != nil {
		return fmt.Errorf("while inserting base environment: %v", err)
	}

	cp.LocalPacker, err = GetLocalPacker(imagePath, b)

	return err
}

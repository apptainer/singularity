// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"context"
	"fmt"
	"os"

	"github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/library"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/build/types"
)

// LibraryConveyorPacker only needs to hold a packer to pack the image it pulls
// as well as extra information about the library it's pulling from
type LibraryConveyorPacker struct {
	b *types.Bundle
	LocalPacker
	ImgCache *cache.ImgCache
}

// SetImgCache sets the image cache to be used for all future operations
func (cp *LibraryConveyorPacker) SetImgCache(imgCache *cache.ImgCache) (err error) {
	cp.ImgCache = imgCache

	return nil
}

// Get downloads container from Singularityhub
func (cp *LibraryConveyorPacker) Get(b *types.Bundle) (err error) {
	sylog.Debugf("Getting container from Library")

	if cp.ImgCache == nil {
		return fmt.Errorf("invalid image cache")
	}

	cp.b = b

	libraryURL := b.Opts.LibraryURL
	authToken := b.Opts.LibraryAuthToken

	if err = makeBaseEnv(cp.b.Rootfs()); err != nil {
		return fmt.Errorf("While inserting base environment: %v", err)
	}

	// check for custom library from definition
	customLib, ok := b.Recipe.Header["library"]
	if ok {
		sylog.Debugf("Using custom library: %v", customLib)
		libraryURL = customLib
	}

	sylog.Debugf("LibraryURL: %v", libraryURL)
	sylog.Debugf("LibraryRef: %v", b.Recipe.Header["from"])

	libraryClient, err := client.NewClient(&client.Config{
		BaseURL:   libraryURL,
		AuthToken: authToken,
	})
	if err != nil {
		return err
	}

	imageRef := library.NormalizeLibraryRef(b.Recipe.Header["from"])

	libraryImage, existOk, err := libraryClient.GetImage(context.TODO(), imageRef)
	if err != nil {
		return fmt.Errorf("while getting image info: %v", err)
	}
	if !existOk {
		return fmt.Errorf("image does not exist in the library: %s", imageRef)
	}

	imageName := uri.GetName("library://" + imageRef)
	imagePath := cp.ImgCache.LibraryImage(libraryImage.Hash, imageName)

	if exists, err := cp.ImgCache.LibraryImageExists(libraryImage.Hash, imageName); err != nil {
		return fmt.Errorf("unable to check if %v exists: %v", imagePath, err)
	} else if !exists {
		sylog.Infof("Downloading library image")

		if err = library.DownloadImageNoProgress(context.TODO(), libraryClient, imagePath, imageRef); err != nil {
			return fmt.Errorf("unable to Download Image: %v", err)
		}

		if cacheFileHash, err := client.ImageHash(imagePath); err != nil {
			return fmt.Errorf("Error getting ImageHash: %v", err)
		} else if cacheFileHash != libraryImage.Hash {
			return fmt.Errorf("Cached File Hash(%s) and Expected Hash(%s) does not match", cacheFileHash, libraryImage.Hash)
		}
	}

	// insert base metadata before unpacking fs
	if err = makeBaseEnv(cp.b.Rootfs()); err != nil {
		return fmt.Errorf("While inserting base environment: %v", err)
	}

	cp.LocalPacker, err = GetLocalPacker(imagePath, cp.b)

	return err
}

// CleanUp removes any tmpfs owned by the conveyorPacker on the filesystem
func (cp *LibraryConveyorPacker) CleanUp() {
	os.RemoveAll(cp.b.Path)
}

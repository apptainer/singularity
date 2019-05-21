// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"fmt"
	"os"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/build/types"
	client "github.com/sylabs/singularity/pkg/client/library"
)

// LibraryConveyorPacker only needs to hold a packer to pack the image it pulls
// as well as extra information about the library it's pulling from
type LibraryConveyorPacker struct {
	b *types.Bundle
	LocalPacker
}

// Get downloads container from Singularityhub
func (cp *LibraryConveyorPacker) Get(b *types.Bundle) (err error) {
	sylog.Debugf("Getting container from Library")

	cp.b = b

	libraryURL := b.Opts.LibraryURL
	authToken := b.Opts.LibraryAuthToken

	if err = makeBaseEnv(cp.b.Rootfs()); err != nil {
		return fmt.Errorf("while inserting base environment: %v", err)
	}

	// check for custom library from definition
	customLib, ok := b.Recipe.Header["library"]
	if ok {
		sylog.Debugf("Using custom library: %v", customLib)
		libraryURL = customLib
	}

	sylog.Debugf("LibraryURL: %v", libraryURL)
	sylog.Debugf("LibraryRef: %v", b.Recipe.Header["from"])

	libURI := "library://" + b.Recipe.Header["from"]
	libraryImage, err := client.GetImage(libraryURL, authToken, libURI)
	if err != nil {
		return err
	}

	imageName := uri.GetName(libURI)
	// Create a cache handle, which will provide access to an existing cache
	// or create a new cache based on the current configuration.
	c, err := cache.NewHandle()
	if c == nil || err != nil {
		return fmt.Errorf("failed to create cache object")
	}

	imagePath, err := c.LibraryImage(libraryImage.Hash, imageName)
	if err != nil {
		return fmt.Errorf("failed to get image path: %s", err)
	}

	exists, err := c.LibraryImageExists(libraryImage.Hash, imageName)
	if err != nil {
		return fmt.Errorf("unable to check if %v with hash %s exists: %v", imagePath, libraryImage.Hash, err)
	} else if !exists {
		sylog.Infof("Downloading library image")
		if err = client.DownloadImage(imagePath, libURI, libraryURL, true, authToken); err != nil {
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
		return fmt.Errorf("while inserting base environment: %v", err)
	}

	cp.LocalPacker, err = GetLocalPacker(imagePath, cp.b)

	return err
}

// CleanUp removes any tmpfs owned by the conveyorPacker on the filesystem
func (cp *LibraryConveyorPacker) CleanUp() {
	os.RemoveAll(cp.b.Path)
}

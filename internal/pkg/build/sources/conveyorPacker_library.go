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
	"github.com/sylabs/singularity/pkg/signing"
	"github.com/sylabs/singularity/pkg/sypgp"
)

// LibraryConveyorPacker only needs to hold a packer to pack the image it pulls
// as well as extra information about the library it's pulling from
type LibraryConveyorPacker struct {
	b *types.Bundle
	LocalPacker
	LibraryURL           string
	AuthToken            string
	AllowUnauthenticated bool
	LocalVerify          bool
}

// Get downloads container from Singularity Library
func (cp *LibraryConveyorPacker) Get(b *types.Bundle) (err error) {
	sylog.Debugf("Getting container from Library")

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

	libURI := "library://" + b.Recipe.Header["from"]
	libraryImage, err := client.GetImage(libraryURL, authToken, libURI)
	if err != nil {
		return err
	}

	imageName := uri.GetName(libURI)
	imagePath := cache.LibraryImage(libraryImage.Hash, imageName)

	if exists, err := cache.LibraryImageExists(libraryImage.Hash, imageName); err != nil {
		return fmt.Errorf("unable to check if %v exists: %v", imagePath, err)
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

	if !cp.AllowUnauthenticated {
		// check if the base container is signed
		imageSigned, err := signing.IsSigned(imagePath, "https://keys.sylabs.io", 0, false, cp.AuthToken, cp.LocalVerify, true)
		if err != nil {
			sylog.Warningf("%v", err)
		}
		if cp.LocalVerify && !imageSigned {
			return fmt.Errorf("unable to build container: no local key matching entity")
		}
		// if its not signed, print a warning
		if !imageSigned {
			sylog.Warningf("The base container is **NOT** signed thus, its content cant be verified!")
			resp, err := sypgp.AskQuestion("Do you really want to continue? [N/y] ")
			if err != nil {
				sylog.Fatalf("Error parsing input: %s", err)
			}
			if resp == "" || resp != "y" && resp != "Y" {
				fmt.Fprintf(os.Stderr, "Stoping build.\n")
				return fmt.Errorf("user said not to build from unsigned container, good choice")
			}
			fmt.Fprintf(os.Stderr, "You can avoid this by using the '-U' flag\n")
		}
	} else {
		sylog.Warningf("Skipping verifction check.")
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

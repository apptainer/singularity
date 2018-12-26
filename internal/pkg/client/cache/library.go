// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	library "github.com/sylabs/singularity/pkg/client/library"
)

const (
	// LibraryDir is the directory inside the cache.Dir where library images are cached
	LibraryDir = "library"
)

// Library returns the directory inside the cache.Dir() where library images are cached
func Library() string {
	return updateCacheSubdir(LibraryDir)
}

// LibraryImage creates a directory inside cache.Dir() with the name of the SHA sum of the image
func LibraryImage(sum, name string) string {
	updateCacheSubdir(filepath.Join(LibraryDir, sum))

	return filepath.Join(Library(), sum, name)
}

// LibraryImageExists returns whether the image with the SHA sum exists in the LibraryImage cache
func LibraryImageExists(sum, name string) (bool, error) {
	_, err := os.Stat(LibraryImage(sum, name))
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// PullLibraryImage is the function that is responsible for pulling an image from a Sylabs library into the cache.
// Requires libraryRef to include library://
func PullLibraryImage(libraryRef, libraryURL string, authToken string) (string, error) {
	libraryImage, err := library.GetImage(libraryURL, authToken, libraryRef)
	if err != nil {
		return "", err
	}

	imageName := uri.GetName(libraryRef)
	imagePath := LibraryImage(libraryImage.Hash, imageName)
	sylog.Debugf("Library Ref: %v", libraryRef)
	sylog.Debugf("Image Name: %v", imageName)
	sylog.Debugf("Image Path: %v", imagePath)

	if exists, err := LibraryImageExists(libraryImage.Hash, imageName); err != nil {
		return "", fmt.Errorf("unable to check if %v exists: %v", imagePath, err)
	} else if !exists {
		sylog.Infof("Downloading library image")
		err := library.DownloadImage(imagePath, libraryRef, libraryURL, true, authToken)
		if err != nil {
			return "", err
		}
	}
	return imagePath, nil
}

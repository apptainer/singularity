// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package libexec

import (
	"io"
	"os"

	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
	net "github.com/sylabs/singularity/pkg/client/net"
	shub "github.com/sylabs/singularity/pkg/client/shub"
)

// PullNetImage is the function that is responsible for pulling an image from http remote url.
func PullNetImage(image, libraryURL string, force bool) {
	err := net.DownloadImage(image, libraryURL, force)
	if err != nil {
		sylog.Fatalf("%v\n", err)
	}
}

// PullLibraryImage is the function that is responsible for pulling an image from a Sylabs library.
func PullLibraryImage(filePath, libraryRef, libraryURL string, force bool, authToken string) {
	cacheImagePath, err := cache.PullLibraryImage(libraryRef, libraryURL, authToken)
	if err != nil {
		sylog.Fatalf("%v\n", err)
	}

	if !force {
		if _, err := os.Stat(filePath); err == nil {
			sylog.Fatalf("image file already exists - will not overwrite")
		}
	}

	// Perms are 777 *prior* to umask
	dstFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		sylog.Fatalf("%v\n", err)
	}
	defer dstFile.Close()

	srcFile, err := os.OpenFile(cacheImagePath, os.O_RDONLY, 0444)
	if err != nil {
		sylog.Fatalf("%v\n", err)
	}
	defer srcFile.Close()

	// Copy SIF from cache
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		sylog.Fatalf("%v\n", err)
	}
}

// PullShubImage is the function that is responsible for pulling an image from a Singularity Hub.
func PullShubImage(filePath, shubRef string, force, noHTTPS bool) {
	err := shub.DownloadImage(filePath, shubRef, force, noHTTPS)
	if err != nil {
		sylog.Fatalf("%v\n", err)
	}
}

// PullOciImage pulls an OCI image to a sif
func PullOciImage(path, uri string, opts types.Options) {
	b, err := build.NewBuild(uri, path, "sif", "", "", opts)
	if err != nil {
		sylog.Fatalf("Unable to pull %v: %v", uri, err)
	}

	if err := b.Full(); err != nil {
		sylog.Fatalf("Unable to pull %v: %v", uri, err)
	}
}

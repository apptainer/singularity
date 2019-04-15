// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package libexec

import (
	"strings"

	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/build/types"
	library "github.com/sylabs/singularity/pkg/client/library"
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
func PullLibraryImage(image, libraryRef, libraryURL string, force bool, authToken string) {
	err := library.DownloadImage(image, libraryRef, libraryURL, force, authToken)
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
func PullOciImage(path, imageURI string, opts types.Options) {
	_, imageName := uri.Split(imageURI)
	if !strings.Contains(imageName, ":") {
		sylog.Infof("No tag specifyed, using ':latest'")
	}

	b, err := build.NewBuild(imageURI, build.Config{Dest: path, Format: "sif", Opts: opts})
	if err != nil {
		sylog.Fatalf("Unable to pull %v: %v", imageURI, err)
	}

	if err := b.Full(); err != nil {
		sylog.Fatalf("Unable to pull %v: %v", imageURI, err)
	}
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package libexec

import (
	library "github.com/singularityware/singularity/src/pkg/library/client"
	shub "github.com/singularityware/singularity/src/pkg/shub/client"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

// PullLibraryImg is the function that is responsible for pulling an image from a Sylabs library.
func PullLibraryImg(image string, libraryRef string, libraryURL string, force bool, authToken string) {
	err := library.DownloadImage(image, libraryRef, libraryURL, force, authToken)
	if err != nil {
		sylog.Fatalf("%v\n", err)
	}
}

// PullShubImg is the function that is responsible for pulling an image from a Singularity Hub.
func PullShubImg(filePath string, shubRef string, force bool) {
	err := shub.DownloadImage(filePath, shubRef, force)
	if err != nil {
		sylog.Fatalf("%v\n", err)
	}
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package libexec

import (
	"github.com/singularityware/singularity/src/pkg/library/client"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

// PullImage is the function that is responsible for pulling an image from a Sylabs library.
func PullImage(image string, library string, libraryURL string, force bool, authToken string) {
	err := client.DownloadImage(image, library, libraryURL, force, authToken)
	if err != nil {
		sylog.Fatalf("%v\n", err)
	}
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package libexec

import (
	"github.com/singularityware/singularity/src/pkg/library/client"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

// PushImage is the function that is responsible for pushing an image to the Sylabs library.
func PushImage(image string, library string, libraryURL string, authToken string) {
	err := client.UploadImage(image, library, libraryURL, authToken)
	if err != nil {
		sylog.Fatalf("%v\n", err)
	}
}

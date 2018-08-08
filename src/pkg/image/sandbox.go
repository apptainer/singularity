// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"fmt"
	"os"
)

// SANDBOX defines constant for directory format
const SANDBOX = 3

type sandboxFormat struct{}

func (f *sandboxFormat) initializer(img *Image, fileinfo os.FileInfo) error {
	if fileinfo.IsDir() {
		img.Type = SANDBOX
		img.Writable = true
	} else {
		return fmt.Errorf("not a directory image")
	}
	return nil
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"fmt"
	"os"
)

// EXT3 defines constants for ext3 format
const EXT3 = 2

type ext3Format struct{}

func (f *ext3Format) initializer(img *Image, fileinfo os.FileInfo) error {
	if fileinfo.IsDir() {
		return fmt.Errorf("not an ext3 image")
	}
	return nil
}

func init() {
	registerFormat("ext3", &ext3Format{})
}

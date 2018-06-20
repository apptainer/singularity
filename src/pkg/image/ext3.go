// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"os"
)

// EXT3 defines constants for ext3 format
const EXT3 = 2

type ext3Format struct {
	file *os.File
}

func (f *ext3Format) Validate(file *os.File) bool {
	f.file = file
	return false
}

func (f *ext3Format) Init(img *Image) error {
	return nil
}

func init() {
	registerFormat(&ext3Format{})
}

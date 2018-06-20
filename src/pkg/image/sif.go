// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"os"
)

type sifFormat struct {
	file *os.File
}

func (f *sifFormat) Validate(file *os.File) bool {
	f.file = file
	return false
}

func (f *sifFormat) Init(img *Image) error {
	return nil
}

func init() {
	registerFormat(&sifFormat{})
}

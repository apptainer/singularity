// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"os"
)

// SANDBOX defines constants for directory format
const SANDBOX = 3

type sandboxFormat struct {
	file *os.File
}

func (f *sandboxFormat) Validate(file *os.File) bool {
	f.file = file
	return false
}

func (f *sandboxFormat) Init(img *Image) error {
	return nil
}

func init() {
	registerFormat(&sandboxFormat{})
}

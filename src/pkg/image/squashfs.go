// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"fmt"
	"os"
)

// SQUASHFS defines constants for squashfs format
const SQUASHFS = 1

type squashfsFormat struct{}

func (f *squashfsFormat) initializer(img *Image, fileinfo os.FileInfo) error {
	if fileinfo.IsDir() {
		return fmt.Errorf("not a squashfs image")
	}
	return nil
}

func init() {
	registerFormat("squashfs", &squashfsFormat{})
}

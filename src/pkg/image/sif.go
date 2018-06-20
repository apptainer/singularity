// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"fmt"
	"os"
)

type sifFormat struct{}

func (f *sifFormat) initializer(img *Image, fileinfo os.FileInfo) error {
	if fileinfo.IsDir() {
		return fmt.Errorf("not a SIF file image")
	}
	return nil
}

func init() {
	registerFormat("sif", &sifFormat{})
}

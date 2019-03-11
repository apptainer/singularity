// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"bytes"
	"fmt"
	"os"
)

// SIF defines constant for sif format
const SIF = 4

const (
	sifMagic = "\x53\x49\x46\x5f\x4d\x41\x47\x49\x43"
)

type sifFormat struct{}

func (f *sifFormat) initializer(img *Image, fileinfo os.FileInfo) error {
	if fileinfo.IsDir() {
		return fmt.Errorf("not a SIF file image")
	}
	b := make([]byte, bufferSize)
	if n, err := img.File.Read(b); err != nil || n != bufferSize {
		return fmt.Errorf("can't read first %d bytes: %s", bufferSize, err)
	}
	if bytes.Index(b, []byte(sifMagic)) == -1 {
		return fmt.Errorf("SIF magic not found")
	}
	img.Type = SIF
	return nil
}

func (f *sifFormat) openMode(writable bool) int {
	if writable {
		return os.O_RDWR
	}
	return os.O_RDONLY
}

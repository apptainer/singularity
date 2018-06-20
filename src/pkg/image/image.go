// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"fmt"
	"os"
)

var registeredFormats = make([]format, 0)

// Image ...
type Image struct {
	Path     string
	Name     string
	Type     string
	File     *os.File
	Offset   uint64
	Writable bool
}

func registerFormat(f format) {
	registeredFormats = append(registeredFormats, f)
}

// format describes the interface that an image format type must implement.
type format interface {
	Validate(*os.File) bool
	Init(*Image) error
}

// Init ...
func Init(path string, writable bool) (*Image, error) {
	flags := os.O_RDONLY
	if writable {
		flags = os.O_RDWR
	}
	file, err := os.OpenFile(path, flags, 0)
	if err != nil {
		return nil, fmt.Errorf("Error while opening image %s: %s", path, err)
	}
	for _, f := range registeredFormats {
		if f.Validate(file) {
			img := &Image{
				Path:     path,
				Name:     file.Name(),
				File:     file,
				Writable: writable,
			}
			if err := f.Init(img); err != nil {
				return nil, err
			}
			return img, nil
		}
	}
	return nil, fmt.Errorf("image format not recognized")
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"fmt"
	"os"
	"path"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

var registeredFormats = make(map[string]format, 0)

// Image ...
type Image struct {
	Path     string
	Name     string
	Type     int
	File     *os.File
	Offset   uint64
	Writable bool
}

func registerFormat(name string, f format) {
	registeredFormats[name] = f
}

// format describes the interface that an image format type must implement.
type format interface {
	initializer(*Image, os.FileInfo) error
}

// Init ...
func Init(filepath string, writable bool) (*Image, error) {
	sylog.Debugf("Entering image format intializer")
	flags := os.O_RDONLY
	if writable {
		flags = os.O_RDWR
	}
	file, err := os.OpenFile(filepath, flags, 0)
	if err != nil {
		return nil, fmt.Errorf("Error while opening image %s: %s", filepath, err)
	}
	img := &Image{
		Path:     filepath,
		Name:     path.Base(filepath),
		File:     file,
		Writable: writable,
	}
	fileinfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	for name, f := range registeredFormats {
		if offset, err := img.File.Seek(0, os.SEEK_SET); err != nil || offset != 0 {
			return nil, err
		}
		if err := f.initializer(img, fileinfo); err == nil {
			return img, nil
		}
		sylog.Debugf("%s format initializer returns: %s", name, err)
	}
	file.Close()
	return nil, fmt.Errorf("image format not recognized")
}

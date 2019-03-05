// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"fmt"
	"io"
	"os"
)

type readerError string

func (e readerError) Error() string { return string(e) }

const (
	// ErrNoSection corresponds to a missing CNI configuration path
	ErrNoSection = readerError("no section found")
	// ErrNoPartition corresponds to a missing CNI plugin path
	ErrNoPartition = readerError("no partition found")
)

func checkImage(image *Image) error {
	if image == nil {
		return fmt.Errorf("image is nil")
	}
	if image.File == nil {
		return fmt.Errorf("image is not open for read")
	}
	return nil
}

func getSectionReader(file *os.File, section Section) io.Reader {
	return io.NewSectionReader(file, int64(section.Offset), int64(section.Size))
}

// NewPartitionReader searches and returns a reader for an image
// partition identified by name or by index, if index is less than 0
// only partition with provided name will be returned if a matching
// entry is found
func NewPartitionReader(image *Image, name string, index int) (io.Reader, error) {
	idx := -1
	if err := checkImage(image); err != nil {
		return nil, err
	}
	if index > 0 {
		l := len(image.Partitions)
		if index > l-1 {
			return nil, fmt.Errorf("index too large, image contains %d partitions", l)
		}
		idx = index
	}
	if name == "" && idx < 0 {
		return nil, fmt.Errorf("no name or positive index provided")
	}
	for i, p := range image.Partitions {
		if p.Name == name || i == idx {
			return getSectionReader(image.File, p), nil
		}
	}
	return nil, ErrNoPartition
}

// NewSectionReader searches and returns a reader for an image
// section identified by name or by index, if index is less than 0
// only section with provided name will be returned if a matching
// entry is found
func NewSectionReader(image *Image, name string, index int) (io.Reader, error) {
	idx := -1
	if err := checkImage(image); err != nil {
		return nil, err
	}
	if index > 0 {
		l := len(image.Sections)
		if index > l-1 {
			return nil, fmt.Errorf("index too large, image contains %d sections", l)
		}
		idx = index
	}
	if name == "" && idx < 0 {
		return nil, fmt.Errorf("no name or positive index provided")
	}
	for i, p := range image.Sections {
		if p.Name == name || i == idx {
			return getSectionReader(image.File, p), nil
		}
	}
	return nil, ErrNoSection
}

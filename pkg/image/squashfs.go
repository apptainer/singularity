// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"unsafe"

	"github.com/sylabs/singularity/internal/pkg/sylog"
)

const (
	squashfsMagic    = "\x68\x73\x71\x73"
	squashfsZlib     = 1
	squashfsLzmaComp = 2
	squashfsLzoComp  = 3
	squashfsXzComp   = 4
	squashfsLz4Comp  = 5
)

// this represents the superblock of a v4 squashfs image
// previous versions of the superblock contain the major and minor versions
// at the same location so we can use this struct to deduce the version
// of the image
type squashfsInfo struct {
	Magic       [4]byte
	Inodes      uint32
	MkfsTime    uint32
	BlockSize   uint32
	Fragments   uint32
	Compression uint16
	BlockLog    uint16
	Flags       uint16
	NoIds       uint16
	Major       uint16
	Minor       uint16
}

type squashfsFormat struct{}

// parseSquashfsHeader de-serialized the squashfs super block from the supplied byte array
// return a struct describing a v4 superblock, the offset of where the superblock began
func parseSquashfsHeader(b []byte) (*squashfsInfo, uint64, error) {
	var offset uint64

	o := bytes.Index(b, []byte(launchString))
	if o > 0 {
		offset += uint64(o + len(launchString) + 1)
	}
	sinfo := &squashfsInfo{}

	if uintptr(offset)+unsafe.Sizeof(sinfo) >= uintptr(len(b)) {
		return nil, offset, fmt.Errorf("can't find squashfs information header")
	}

	buffer := bytes.NewReader(b[offset:])

	if err := binary.Read(buffer, binary.LittleEndian, sinfo); err != nil {
		return nil, offset, fmt.Errorf("can't read the top of the image")
	}
	if !bytes.Equal(sinfo.Magic[:], []byte(squashfsMagic)) {
		return nil, offset, fmt.Errorf("not a valid squashfs image")
	}

	return sinfo, offset, nil
}

// CheckSquashfsHeader checks if byte content contains a valid squashfs header
// and returns offset where squashfs partition starts
func CheckSquashfsHeader(b []byte) (uint64, error) {
	sinfo, offset, err := parseSquashfsHeader(b)
	if err != nil {
		return offset, fmt.Errorf("while parsing squashfs super block: %v", err)
	}

	if sinfo.Compression != squashfsZlib {
		compressionType := ""
		switch sinfo.Compression {
		case squashfsLzmaComp:
			compressionType = "lzma"
		case squashfsLz4Comp:
			compressionType = "lz4"
		case squashfsLzoComp:
			compressionType = "lzo"
		case squashfsXzComp:
			compressionType = "xz"
		}
		sylog.Infof("squashfs image was compressed with %s, if it failed to run, please contact image's author", compressionType)
	}
	return offset, nil
}

// GetSquashfsComp checks if byte content contains a valid squashfs header
// and returns type of compression used
func GetSquashfsComp(b []byte) (string, error) {
	sb, _, err := parseSquashfsHeader(b)
	if err != nil {
		return "", fmt.Errorf("while parsing squashfs super block: %v", err)
	}

	// tighten up this check to at least look a the major version
	if sb.Major == 4 {
		var compType string
		switch sb.Compression {
		case squashfsZlib:
			compType = "gzip"
		case squashfsLzmaComp:
			compType = "lzma"
		case squashfsLz4Comp:
			compType = "lz4"
		case squashfsLzoComp:
			compType = "lzo"
		case squashfsXzComp:
			compType = "xz"
		}
		return compType, nil
	} else if sb.Major < 4 {
		// v3 and eariler super blocks always use gzip comp
		// different compressors were introduced after the change
		// to v4 super blocks
		return "gzip", nil
	}

	return "", fmt.Errorf("not a valid squashfs image")
}

func (f *squashfsFormat) initializer(img *Image, fileinfo os.FileInfo) error {
	if fileinfo.IsDir() {
		return fmt.Errorf("not a squashfs image")
	}
	b := make([]byte, bufferSize)
	if n, err := img.File.Read(b); err != nil || n != bufferSize {
		return fmt.Errorf("can't read first %d bytes: %s", bufferSize, err)
	}
	offset, err := CheckSquashfsHeader(b)
	if err != nil {
		return err
	}
	img.Type = SQUASHFS
	img.Partitions = []Section{
		{
			Offset: offset,
			Size:   uint64(fileinfo.Size()) - offset,
			Type:   SQUASHFS,
			Name:   RootFs,
		},
	}

	if img.Writable {
		sylog.Warningf("squashfs is not a writable filesystem")
		img.Writable = false
	}

	return nil
}

func (f *squashfsFormat) openMode(writable bool) int {
	return os.O_RDONLY
}

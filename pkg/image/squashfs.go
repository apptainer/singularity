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

type squashfsInfo struct {
	Magic       [4]byte
	Inodes      uint32
	MkfsTime    uint32
	BlockSize   uint32
	Fragments   uint32
	Compression uint16
}

type squashfsFormat struct{}

// CheckSquashfsHeader checks if byte content contains a valid squashfs header
// and returns offset where squashfs partition start
func CheckSquashfsHeader(b []byte) (uint64, error) {
	var offset uint64

	o := bytes.Index(b, []byte(launchString))
	if o > 0 {
		offset += uint64(o + len(launchString) + 1)
	}
	sinfo := &squashfsInfo{}

	if uintptr(offset)+unsafe.Sizeof(sinfo) >= uintptr(len(b)) {
		return offset, fmt.Errorf("can't find squashfs information header")
	}

	buffer := bytes.NewReader(b[offset:])

	if err := binary.Read(buffer, binary.LittleEndian, sinfo); err != nil {
		return offset, fmt.Errorf("can't read the top of the image")
	}
	if bytes.Compare(sinfo.Magic[:], []byte(squashfsMagic)) != 0 {
		return offset, fmt.Errorf("not a valid squashfs image")
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
	img.Partitions[0].Offset = offset
	img.Partitions[0].Size = uint64(fileinfo.Size()) - offset
	img.Partitions[0].Type = SQUASHFS
	img.Partitions[0].Name = RootFs

	if img.Writable {
		sylog.Warningf("squashfs is not a writable filesystem")
		img.Writable = false
	}

	return nil
}

func (f *squashfsFormat) openMode(writable bool) int {
	return os.O_RDONLY
}

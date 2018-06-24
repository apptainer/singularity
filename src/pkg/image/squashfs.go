// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"unsafe"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

// SQUASHFS defines constant for squashfs format
const SQUASHFS = 1

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

func (f *squashfsFormat) initializer(img *Image, fileinfo os.FileInfo) error {
	if fileinfo.IsDir() {
		return fmt.Errorf("not a squashfs image")
	}
	b := make([]byte, bufferSize)
	if n, err := img.File.Read(b); err != nil || n != bufferSize {
		return fmt.Errorf("can't read first %d bytes: %s", bufferSize, err)
	}
	offset := 0
	o := bytes.Index(b, []byte(launchString))
	if o > 0 {
		offset += o + len(launchString) + 1
	}
	sinfo := &squashfsInfo{}

	if uintptr(offset)+unsafe.Sizeof(sinfo) > bufferSize {
		return fmt.Errorf("can't find squashfs information header")
	}

	buffer := bytes.NewReader(b[offset:])

	if err := binary.Read(buffer, binary.LittleEndian, sinfo); err != nil {
		return fmt.Errorf("can't read the top of the image")
	}
	if bytes.Compare(sinfo.Magic[:], []byte(squashfsMagic)) != 0 {
		return fmt.Errorf("not a valid squashfs image")
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
	img.Type = SQUASHFS
	img.Offset = uint64(offset)
	img.Size = uint64(fileinfo.Size()) - img.Offset
	return nil
}

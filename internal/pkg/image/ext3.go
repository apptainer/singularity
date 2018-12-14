// Copyright (c) 2018, Sylabs Inc. All rights reserved.
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
)

// EXT3 defines constant for ext3 format
const EXT3 = 2

const (
	extMagicOffset      = 1080
	extMagic            = "\x53\xEF"
	compatHasJournal    = 0x4
	incompatFileType    = 0x2
	incompatRecover     = 0x4
	incompatMetabg      = 0x10
	rocompatSparseSuper = 0x1
	rocompatLargeFile   = 0x2
	rocompatBtreeDir    = 0x4
)

const notValidExt3ImageMessage = "file is not a valid ext3 image"

type extFSInfo struct {
	Magic    [2]byte
	State    uint16
	Dummy    [8]uint32
	Compat   uint32
	Incompat uint32
	Rocompat uint32
}

type ext3Format struct{}

// CheckExt3Header checks if byte content contains a valid ext3 header
// and returns offset where ext3 partition begin
func CheckExt3Header(b []byte) (uint64, error) {
	var offset uint64 = extMagicOffset

	o := bytes.Index(b, []byte(launchString))
	if o > 0 {
		offset += uint64(o + len(launchString) + 1)
	}
	einfo := &extFSInfo{}

	if uintptr(offset)+unsafe.Sizeof(einfo) >= uintptr(len(b)) {
		return offset, fmt.Errorf("can't find ext3 information header")
	}
	buffer := bytes.NewReader(b[offset:])

	if err := binary.Read(buffer, binary.LittleEndian, einfo); err != nil {
		return offset, fmt.Errorf("can't read the top of the image")
	}
	if bytes.Compare(einfo.Magic[:], []byte(extMagic)) != 0 {
		return offset, fmt.Errorf(notValidExt3ImageMessage)
	}
	if einfo.Compat&compatHasJournal == 0 {
		return offset, fmt.Errorf(notValidExt3ImageMessage)
	}
	if einfo.Incompat&^(incompatFileType|incompatRecover|incompatMetabg) != 0 {
		return offset, fmt.Errorf(notValidExt3ImageMessage)
	}
	if einfo.Rocompat&^(rocompatSparseSuper|rocompatLargeFile|rocompatBtreeDir) != 0 {
		return offset, fmt.Errorf(notValidExt3ImageMessage)
	}
	offset -= extMagicOffset
	return offset, nil
}

func (f *ext3Format) initializer(img *Image, fileinfo os.FileInfo) error {
	if fileinfo.IsDir() {
		return fmt.Errorf("not an ext3 image")
	}
	b := make([]byte, bufferSize)
	if n, err := img.File.Read(b); err != nil || n != bufferSize {
		return fmt.Errorf("can't read first %d bytes: %s", bufferSize, err)
	}
	offset, err := CheckExt3Header(b)
	if err != nil {
		return err
	}
	img.Type = EXT3
	img.Offset = offset
	img.Size = uint64(fileinfo.Size()) - img.Offset
	return nil
}

func (f *ext3Format) openMode(writable bool) int {
	if writable {
		return os.O_RDWR
	}
	return os.O_RDONLY
}

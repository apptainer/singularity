// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"bytes"
	"fmt"
	"os"

	"github.com/sylabs/sif/pkg/sif"
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

	// Load the SIF file
	fimg, err := sif.LoadContainerFp(img.File, !img.Writable)
	if err != nil {
		return err
	}

	// Get the default system partition image
	part, _, err := fimg.GetPartPrimSys()
	if err != nil {
		return err
	}

	// record the fs type
	fstype, err := part.GetFsType()
	if err != nil {
		return err
	}
	if fstype == sif.FsSquash {
		img.Partitions[0].Type = SQUASHFS
	} else if fstype == sif.FsExt3 {
		img.Partitions[0].Type = EXT3
	} else {
		return fmt.Errorf("unknown file system type: %v", fstype)
	}

	img.Partitions[0].Offset = uint64(part.Fileoff)
	img.Partitions[0].Size = uint64(part.Filelen)

	// check if there is overlay partitions
	// Get descriptor for group associated with system partition
	descriptors, _, err := fimg.GetPartFromGroup(part.Groupid)
	if err != nil {
		return err
	}
	// Determine if an overlay partition exists
	for _, desc := range descriptors {
		ptype, err := desc.GetPartType()
		if err != nil {
			return err
		}
		if ptype == sif.PartOverlay {
			fstype, err := desc.GetFsType()
			if err != nil {
				continue
			}
			partition := Partition{
				Offset: uint64(desc.Fileoff),
				Size:   uint64(desc.Filelen),
			}
			switch fstype {
			case sif.FsSquash:
				partition.Type = SQUASHFS
			case sif.FsExt3:
				partition.Type = EXT3
			}
			img.Partitions = append(img.Partitions, partition)
		}
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

// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"syscall"

	"github.com/sylabs/sif/pkg/sif"
)

const (
	sifMagic = "\x53\x49\x46\x5f\x4d\x41\x47\x49\x43"
)

type sifFormat struct{}

func checkPartitionType(img *Image, fstype sif.Fstype, offset int64) (uint32, error) {
	header := make([]byte, bufferSize)

	if _, err := img.File.ReadAt(header, offset); err != nil {
		return 0, fmt.Errorf("failed to read SIF partition at offset %d: %s", offset, err)
	}

	switch fstype {
	case sif.FsSquash:
		if _, err := CheckSquashfsHeader(header[:]); err != nil {
			return 0, fmt.Errorf("error while checking squashfs header: %s", err)
		}
		return SQUASHFS, nil
	case sif.FsExt3:
		if _, err := CheckExt3Header(header[:]); err != nil {
			return 0, fmt.Errorf("error while checking ext3 header: %s", err)
		}
		return EXT3, nil
	case sif.FsEncryptedSquashfs:
		return ENCRYPTSQUASHFS, nil
	}

	return 0, fmt.Errorf("unknown filesystem type %v", fstype)
}

func (f *sifFormat) initializer(img *Image, fileinfo os.FileInfo) error {
	if fileinfo.IsDir() {
		return debugError("not a sif file image")
	}
	b := make([]byte, bufferSize)
	if n, err := img.File.Read(b); err != nil || n != bufferSize {
		return debugErrorf("can't read first %d bytes: %s", bufferSize, err)
	}
	if !bytes.Contains(b, []byte(sifMagic)) {
		return debugError("SIF magic not found")
	}

	// Load the SIF file
	fimg, err := sif.LoadContainerFp(img.File, !img.Writable)
	if err != nil {
		return err
	}

	// Check the compatibility of the image's target architecture
	// TODO: we should check if we need to deal with compatible architectures:
	// For example, i386 can run on amd64 and maybe some ARM processor can run <= armv6 instructions
	// on asasrch64 (someone should double check).
	// TODO: The typically workflow is:
	// 1. pull image from docker/library/shub (pull/build commands)
	// 2. extract image file system to temp folder (build commands)
	// 3. if definition file contains a 'executable' section, the architecture check should
	// occur (or delegate to runtime which would fail during execution).
	// The current code will be called by the starter which will cover most of the
	// workflow described above. However, SIF is currently build upon the assumption
	// that the architecture is assigned based on the architecture defined by a Go
	// runtime, which is not 100% compliant with the intended workflow.
	sifArch := string(fimg.Header.Arch[:sif.HdrArchLen-1])
	if sifArch != sif.HdrArchUnknown && sifArch != sif.GetSIFArch(runtime.GOARCH) {
		return fmt.Errorf("the image's architecture (%s) is incompatible with the host's (%s)", sif.GetGoArch(sifArch), runtime.GOARCH)
	}

	groupID := -1

	// Get the default system partition image
	for _, desc := range fimg.DescrArr {
		if !desc.Used {
			continue
		}
		if desc.Datatype != sif.DataPartition {
			continue
		}
		ptype, err := desc.GetPartType()
		if err != nil {
			continue
		}
		if ptype != sif.PartPrimSys {
			continue
		}
		fstype, err := desc.GetFsType()
		if err != nil {
			continue
		}

		// checks if the partition length is greater that the file
		// size which may reveal a corrupted image (see issue #3996)
		if fimg.Filesize < desc.Filelen+desc.Fileoff {
			return fmt.Errorf("SIF image %s is corrupted: wrong partition size", img.File.Name())
		}

		htype, err := checkPartitionType(img, fstype, desc.Fileoff)
		if err != nil {
			return fmt.Errorf("while checking system partition header: %s", err)
		}

		img.Partitions = []Section{
			{
				Offset: uint64(desc.Fileoff),
				Size:   uint64(desc.Filelen),
				Name:   RootFs,
				Type:   htype,
			},
		}

		groupID = int(desc.Groupid)
		break
	}

	for _, desc := range fimg.DescrArr {
		if !desc.Used {
			continue
		}
		if ptype, err := desc.GetPartType(); err == nil {
			// exclude partitions that are not types data or overlay
			if ptype != sif.PartData && ptype != sif.PartOverlay {
				continue
			}
			// ignore overlay partitions not associated to root
			// filesystem group ID
			if ptype == sif.PartOverlay && groupID != int(desc.Groupid) {
				continue
			}
			fstype, err := desc.GetFsType()
			if err != nil {
				continue
			}

			if fimg.Filesize < desc.Filelen+desc.Fileoff {
				return fmt.Errorf("SIF image %s is corrupted: wrong partition size", img.File.Name())
			}

			htype, err := checkPartitionType(img, fstype, desc.Fileoff)
			if err != nil {
				return fmt.Errorf("while checking data partition header: %s", err)
			}

			partition := Section{
				Offset: uint64(desc.Fileoff),
				Size:   uint64(desc.Filelen),
				Name:   desc.GetName(),
				Type:   htype,
			}
			img.Partitions = append(img.Partitions, partition)
		} else if desc.Datatype != 0 {
			data := Section{
				Offset: uint64(desc.Fileoff),
				Size:   uint64(desc.Filelen),
				Type:   uint32(desc.Datatype),
				Name:   desc.GetName(),
			}
			img.Sections = append(img.Sections, data)
		}
	}

	img.Type = SIF

	// UnloadContainer close image, just want to unmap image
	// from memory
	if !fimg.Amodebuf {
		if err := syscall.Munmap(fimg.Filedata); err != nil {
			return fmt.Errorf("while calling unmapping SIF file")
		}
	}

	return nil
}

func (f *sifFormat) openMode(writable bool) int {
	if writable {
		return os.O_RDWR
	}
	return os.O_RDONLY
}

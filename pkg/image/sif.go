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
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

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
	if !bytes.Contains(b, []byte(sifMagic)) {
		return fmt.Errorf("SIF magic not found")
	}

	// Load the SIF file
	fimg, err := sif.LoadContainerFp(img.File, !img.Writable)
	if err != nil {
		return err
	}

	// Check the compatibility of the image's target architecture
	// TODO: we should check if we need to deal with compatible architectures. For example, i386 can run on amd64 and maybe some ARM processor can run <= armv6 instructions on asasrch64 (someone should double check).
	// TODO: The typically workflow is:
	// 1. pull image from docker/library/shub (pull/build commands)
	// 2. extract image file system to temp folder (build commands)
	// 3. if definition file contains a 'executable' section, the architecture check should occur (or delegate to runtime which would fail during execution).
	// The current code will be called by the started which will cover most of the workflow desribed above. However, SIF is currently build upon the assumption that the architecture is assigned based on the architecture defined by a Go runtime, which is not 100% compliant with the intended workflow.
	sifArch := string(fimg.Header.Arch[:sif.HdrArchLen-1])
	if sifArch != sif.HdrArchUnknown && sifArch != sif.GetSIFArch(runtime.GOARCH) {
		sylog.Fatalf("the image's architecture (%s) is incompatible with the host (%s)", sif.GetGoArch(sifArch), runtime.GOARCH)
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

		sylog.Debugf("Fs type is %s", fstype)

		img.Partitions = []Section{
			{
				Offset: uint64(desc.Fileoff),
				Size:   uint64(desc.Filelen),
				Name:   RootFs,
			},
		}

		if fstype == sif.FsSquash {
			img.Partitions[0].Type = SQUASHFS
		} else if fstype == sif.FsExt3 {
			img.Partitions[0].Type = EXT3
		} else if fstype == sif.FsEncrypt {
			img.Partitions[0].Type = ENCRYPTFS
		} else {
			return fmt.Errorf("unknown file system type: %v", fstype)
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
			partition := Section{
				Offset: uint64(desc.Fileoff),
				Size:   uint64(desc.Filelen),
				Name:   desc.GetName(),
			}
			switch fstype {
			case sif.FsSquash:
				partition.Type = SQUASHFS
			case sif.FsExt3:
				partition.Type = EXT3
			default:
				partition.Type = uint32(fstype)
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

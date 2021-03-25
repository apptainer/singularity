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
	"github.com/sylabs/singularity/internal/pkg/util/machine"
)

const (
	//SIFDescOCIConfigJSON is the name of the SIF descriptor holding the OCI configuration.
	SIFDescOCIConfigJSON = "oci-config.json"
	// SIFDescInspectMetadataJSON is the name of the SIF descriptor holding the container metadata.
	SIFDescInspectMetadataJSON = "inspect-metadata.json"
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
	case sif.FsRaw:
		return RAW, nil
	}

	return 0, fmt.Errorf("unknown filesystem type %v", fstype)
}

func (f *sifFormat) initializer(img *Image, fi os.FileInfo) error {
	if fi.IsDir() {
		return debugError("not a sif file image")
	}
	b := make([]byte, bufferSize)
	if n, err := img.File.Read(b); err != nil || n != bufferSize {
		return debugErrorf("can't read first %d bytes: %v", bufferSize, err)
	}
	if !bytes.Contains(b, []byte(sif.HdrMagic)) {
		return debugError("SIF magic not found")
	}

	// Load the SIF file
	fimg, err := sif.LoadContainerFp(img.File, !img.Writable)
	if err != nil {
		return err
	}

	groupID := -1

	// Get the default system partition image
	for _, desc := range fimg.DescrArr {
		if !desc.Used {
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

		// Check the compatibility of the image's target architecture, the
		// CompatibleWith call will also check that the current machine
		// has persistent emulation enabled in /proc/sys/fs/binfmt_misc to
		// be able to execute container process correctly
		sifArch := string(fimg.Header.Arch[:sif.HdrArchLen-1])
		goArch := sif.GetGoArch(sifArch)
		if sifArch != sif.HdrArchUnknown && !machine.CompatibleWith(goArch) {
			return fmt.Errorf("the image's architecture (%s) could not run on the host's (%s)", goArch, runtime.GOARCH)
		}

		img.Partitions = []Section{
			{
				Offset:       uint64(desc.Fileoff),
				Size:         uint64(desc.Filelen),
				ID:           desc.ID,
				Name:         RootFs,
				Type:         htype,
				AllowedUsage: RootFsUsage,
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
			// ignore overlay partitions not associated to root filesystem group ID if any
			if ptype == sif.PartOverlay && groupID > 0 && groupID != int(desc.Groupid) {
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

			var usage Usage

			if ptype == sif.PartOverlay {
				usage = OverlayUsage
			} else {
				usage = DataUsage
			}

			partition := Section{
				Offset:       uint64(desc.Fileoff),
				Size:         uint64(desc.Filelen),
				ID:           desc.ID,
				Name:         desc.GetName(),
				Type:         htype,
				AllowedUsage: usage,
			}
			img.Partitions = append(img.Partitions, partition)
			img.Usage |= usage
		} else if desc.Datatype != 0 {
			data := Section{
				Offset:       uint64(desc.Fileoff),
				Size:         uint64(desc.Filelen),
				ID:           desc.ID,
				Type:         uint32(desc.Datatype),
				Name:         desc.GetName(),
				AllowedUsage: DataUsage,
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

func (f *sifFormat) lock(img *Image) error {
	for _, part := range img.Partitions {
		if part.Type != EXT3 {
			continue
		}
		if err := lockSection(img, part); err != nil {
			return fmt.Errorf("while locking ext3 partition from %s: %s", img.Path, err)
		}
	}
	return nil
}

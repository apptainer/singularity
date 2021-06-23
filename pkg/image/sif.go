// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"bytes"
	"fmt"
	"os"
	"runtime"

	"github.com/hpcng/sif/v2/pkg/sif"
	"github.com/hpcng/singularity/internal/pkg/util/machine"
)

const (
	// SIFDescOCIConfigJSON is the name of the SIF descriptor holding the OCI configuration.
	SIFDescOCIConfigJSON = "oci-config.json"
	// SIFDescInspectMetadataJSON is the name of the SIF descriptor holding the container metadata.
	SIFDescInspectMetadataJSON = "inspect-metadata.json"
)

type sifFormat struct{}

func checkPartitionType(img *Image, fstype sif.FSType, offset int64) (uint32, error) {
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
	if !bytes.Contains(b, []byte("SIF_MAGIC")) {
		return debugError("SIF magic not found")
	}

	flag := os.O_RDONLY
	if img.Writable {
		flag = os.O_RDWR
	}

	// Load the SIF file
	fimg, err := sif.LoadContainer(img.File,
		sif.OptLoadWithFlag(flag),
		sif.OptLoadWithCloseOnUnload(false),
	)
	if err != nil {
		return err
	}
	defer fimg.UnloadContainer()

	var groupID uint32

	// Get the default system partition image
	desc, err := fimg.GetDescriptor(sif.WithPartitionType(sif.PartPrimSys))
	if err == nil {
		fstype, _, goArch, err := desc.PartitionMetadata()
		if err != nil {
			return err
		}

		htype, err := checkPartitionType(img, fstype, desc.Offset())
		if err != nil {
			return fmt.Errorf("while checking system partition header: %s", err)
		}

		// Check the compatibility of the image's target architecture, the
		// CompatibleWith call will also check that the current machine
		// has persistent emulation enabled in /proc/sys/fs/binfmt_misc to
		// be able to execute container process correctly
		if goArch != "unknown" && !machine.CompatibleWith(goArch) {
			return fmt.Errorf("the image's architecture (%s) could not run on the host's (%s)", goArch, runtime.GOARCH)
		}

		groupID = desc.GroupID()

		img.Partitions = []Section{
			{
				Offset:       uint64(desc.Offset()),
				Size:         uint64(desc.Size()),
				ID:           desc.ID(),
				Name:         RootFs,
				Type:         htype,
				AllowedUsage: RootFsUsage,
			},
		}
	}

	fimg.WithDescriptors(func(desc sif.Descriptor) bool {
		if fstype, ptype, _, err := desc.PartitionMetadata(); err == nil {
			// exclude partitions that are not types data or overlay
			if ptype != sif.PartData && ptype != sif.PartOverlay {
				return false
			}
			// ignore overlay partitions not associated to root filesystem group ID if any
			if ptype == sif.PartOverlay && groupID > 0 && groupID != desc.GroupID() {
				return false
			}

			htype, err := checkPartitionType(img, fstype, desc.Offset())
			if err != nil {
				return false
			}

			var usage Usage

			if ptype == sif.PartOverlay {
				usage = OverlayUsage
			} else {
				usage = DataUsage
			}

			partition := Section{
				Offset:       uint64(desc.Offset()),
				Size:         uint64(desc.Size()),
				ID:           desc.ID(),
				Name:         desc.Name(),
				Type:         htype,
				AllowedUsage: usage,
			}
			img.Partitions = append(img.Partitions, partition)
			img.Usage |= usage
		} else if desc.DataType() != 0 {
			data := Section{
				Offset:       uint64(desc.Offset()),
				Size:         uint64(desc.Size()),
				ID:           desc.ID(),
				Type:         uint32(desc.DataType()),
				Name:         desc.Name(),
				AllowedUsage: DataUsage,
			}
			img.Sections = append(img.Sections, data)
		}
		return false
	})

	img.Type = SIF

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

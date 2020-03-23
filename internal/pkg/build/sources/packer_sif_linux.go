// Copyright (c) 2019-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/image/unpacker"
	"github.com/sylabs/singularity/pkg/sylog"
)

// Pack puts relevant objects in a Bundle.
func (p *SIFPacker) Pack(context.Context) (*types.Bundle, error) {
	err := unpackSIF(p.b, p.img)
	if err != nil {
		sylog.Errorf("unpackSIF failed: %s", err)
		return nil, err
	}

	return p.b, nil
}

// unpackSIF parses through the sif file and places each component
// in the sandbox. First pass just assumes a single system partition,
// later passes will handle more complex sif files.
func unpackSIF(b *types.Bundle, img *image.Image) (err error) {
	part, err := img.GetRootFsPartition()
	if err != nil {
		return fmt.Errorf("while getting root filesystem in %s: %s", img.Name, err)
	}

	switch part.Type {
	case image.SQUASHFS:
		// create a reader for rootfs partition
		reader, err := image.NewPartitionReader(img, "", 0)
		if err != nil {
			return fmt.Errorf("could not extract root filesystem: %s", err)
		}

		s := unpacker.NewSquashfs()

		// extract root filesystem
		if err := s.ExtractAll(reader, b.RootfsPath); err != nil {
			return fmt.Errorf("root filesystem extraction failed: %s", err)
		}
	case image.EXT3:

		// extract ext3 partition by mounting
		sylog.Debugf("Ext3 partition detected, mounting to extract.")
		if err := unpackExt3(b, img); err != nil {
			return fmt.Errorf("while copying partition data to bundle: %v", err)
		}
	default:
		return fmt.Errorf("unrecognized partition format")
	}

	ociReader, err := image.NewSectionReader(img, image.SIFDescOCIConfigJSON, -1)
	if err == image.ErrNoSection {
		sylog.Debugf("No %s section found", image.SIFDescOCIConfigJSON)
	} else if err != nil {
		return fmt.Errorf("could not get OCI config section reader: %v", err)
	} else {
		ociConfig, err := ioutil.ReadAll(ociReader)
		if err != nil {
			return fmt.Errorf("could not read OCI config: %v", err)
		}
		b.JSONObjects[image.SIFDescOCIConfigJSON] = ociConfig
	}
	return nil
}

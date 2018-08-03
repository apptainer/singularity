// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/singularityware/singularity/src/pkg/build/types"
	"github.com/singularityware/singularity/src/pkg/image"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/loop"
)

// LocalConveyor only needs to hold the conveyor to have the needed data to pack
type LocalConveyor struct {
	src   string
	tmpfs string
}

type localPacker interface {
	Pack() (*types.Bundle, error)
}

// LocalConveyorPacker only needs to hold the conveyor to have the needed data to pack
type LocalConveyorPacker struct {
	LocalConveyor
	localPacker
}

func getLocalPacker(src, tmpfs string) (localPacker, error) {
	imageObject, err := image.Init(src, false)
	if err != nil {
		return nil, err
	}

	info := new(loop.Info64)

	switch imageObject.Type {
	case image.SIF:
		sylog.Debugf("Packing from SIF")

		return &SIFPacker{
			srcfile: src,
			tmpfs:   tmpfs,
		}, nil
	case image.SQUASHFS:
		sylog.Debugf("Packing from Squashfs")

		info.Offset = imageObject.Offset
		info.SizeLimit = imageObject.Size

		return &SquashfsPacker{
			srcfile: src,
			tmpfs:   tmpfs,
			info:    info,
		}, nil
	case image.EXT3:
		sylog.Debugf("Packing from Ext3")

		info.Offset = imageObject.Offset
		info.SizeLimit = imageObject.Size

		return &Ext3Packer{
			srcfile: src,
			tmpfs:   tmpfs,
			info:    info,
		}, nil
	case image.SANDBOX:
		sylog.Debugf("Packing from Sandbox")

		return &SandboxPacker{
			srcdir: src,
			tmpfs:  tmpfs,
		}, nil
	default:
		return nil, fmt.Errorf("invalid image format")
	}
}

// Get just stores the source
func (c *LocalConveyorPacker) Get(recipe types.Definition) (err error) {
	c.src = filepath.Clean(recipe.Header["from"])

	c.tmpfs, err = ioutil.TempDir("", "temp-local-")
	if err != nil {
		return
	}

	c.localPacker, err = getLocalPacker(c.src, c.tmpfs)
	return err
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"io/ioutil"

	"github.com/singularityware/singularity/src/pkg/image"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/loop"
)

// LocalConveyor only needs to hold the conveyor to have the needed data to pack
type LocalConveyor struct {
	src   string
	tmpfs string
}

// LocalPacker only needs to hold the data needed to pack
type LocalPacker struct {
	src   string
	tmpfs string
}

// LocalConveyorPacker only needs to hold the conveyor to have the needed data to pack
type LocalConveyorPacker struct {
	LocalConveyor
	lp LocalPacker
}

// Get just stores the source
func (c *LocalConveyor) Get(recipe Definition) (err error) {
	c.src = recipe.Header["from"]

	c.tmpfs, err = ioutil.TempDir("", "temp-local-")
	if err != nil {
		return
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *LocalPacker) Pack() (b *Bundle, err error) {
	var p Packer
	rootfs := cp.src

	imageObject, err := image.Init(rootfs, false)
	if err != nil {
		return nil, err
	}

	info := new(loop.Info64)

	switch imageObject.Type {
	case image.SIF:
		sylog.Fatalf("Building from SIF not yet supported")

		// Not yet implemented
		// imageObject.Offset = uint64(part.Fileoff)
		// imageObject.Size = uint64(part.Filelen)
		// info.Offset = imageObject.Offset
		// info.SizeLimit = imageObject.Size
	case image.SQUASHFS:
		sylog.Debugf("Packing from Squashfs")

		info.Offset = imageObject.Offset
		info.SizeLimit = imageObject.Size

		p = &SquashfsPacker{
			srcfile: rootfs,
			tmpfs:   cp.tmpfs,
			info:    info,
		}
	case image.EXT3:
		sylog.Debugf("Packing from Ext3")

		info.Offset = imageObject.Offset
		info.SizeLimit = imageObject.Size

		p = &Ext3Packer{
			srcfile: rootfs,
			tmpfs:   cp.tmpfs,
			info:    info,
		}
	case image.SANDBOX:
		sylog.Debugf("Packing from Sandbox")

		p = &SandboxPacker{
			srcdir: rootfs,
			tmpfs:  cp.tmpfs,
		}
	default:
		sylog.Fatalf("Invalid image format from shub")
	}

	b, err = p.Pack()
	if err != nil {
		sylog.Errorf("Local Pack failed", err.Error())
		return nil, err
	}

	b.Recipe = Definition{}

	return b, nil
}

// Pack puts relevant objects in a Bundle!
func (cp *LocalConveyorPacker) Pack() (b *Bundle, err error) {

	cp.lp = LocalPacker{cp.src, cp.tmpfs}

	b, err = cp.lp.Pack()
	if err != nil {
		sylog.Errorf("Local Pack failed", err.Error())
		return nil, err
	}

	b.Recipe = Definition{}

	return b, nil
}

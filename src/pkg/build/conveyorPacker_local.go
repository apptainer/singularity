// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"io/ioutil"
	"path/filepath"

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
	c.src = filepath.Clean(recipe.Header["from"])

	c.tmpfs, err = ioutil.TempDir("", "temp-local-")
	if err != nil {
		return
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *LocalPacker) Pack() (b *Bundle, err error) {
	var p Packer

	imageObject, err := image.Init(cp.src, false)
	if err != nil {
		return nil, err
	}

	info := new(loop.Info64)

	switch imageObject.Type {
	case image.SIF:
		sylog.Debugf("Packing from SIF")

		p = &SIFPacker{
			srcfile: cp.src,
			tmpfs:   cp.tmpfs,
		}
	case image.SQUASHFS:
		sylog.Debugf("Packing from Squashfs")

		info.Offset = imageObject.Offset
		info.SizeLimit = imageObject.Size

		p = &SquashfsPacker{
			srcfile: cp.src,
			tmpfs:   cp.tmpfs,
			info:    info,
		}
	case image.EXT3:
		sylog.Debugf("Packing from Ext3")

		info.Offset = imageObject.Offset
		info.SizeLimit = imageObject.Size

		p = &Ext3Packer{
			srcfile: cp.src,
			tmpfs:   cp.tmpfs,
			info:    info,
		}
	case image.SANDBOX:
		sylog.Debugf("Packing from Sandbox")

		p = &SandboxPacker{
			srcdir: cp.src,
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

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// Copyright (c) 2018, Vanessa Sochat. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

/*
#include <unistd.h>
#include "image/image.h"
#include "util/config_parser.h"
*/
// #cgo CFLAGS: -I../../runtime/c/lib
// #cgo LDFLAGS: -L../../../builddir/lib -lruntime -luuid
import "C"
import (
	"io/ioutil"

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

	//leverage C code to properly get image information
	C.singularity_config_init()

	imageObject := C.singularity_image_init(C.CString(rootfs), 0)

	info := new(loop.Info64)

	switch C.singularity_image_type(&imageObject) {
	case 1:
		//squashfs
		sylog.Debugf("Packing from Squashfs")

		info.Offset = uint64(C.uint(imageObject.offset))
		info.SizeLimit = uint64(C.uint(imageObject.size))

		p = &SquashfsPacker{
			srcfile: rootfs,
			tmpfs:   cp.tmpfs,
			info:    info,
		}
	case 2:
		//ext3
		sylog.Debugf("Packing from Ext3")

		info.Offset = uint64(C.uint(imageObject.offset))
		info.SizeLimit = uint64(C.uint(imageObject.size))

		p = &Ext3Packer{
			srcfile: rootfs,
			tmpfs:   cp.tmpfs,
			info:    info,
		}
	case 3:
		//sandbox
		sylog.Debugf("Packing from Sandbox")

		p = &SandboxPacker{
			srcdir: rootfs,
			tmpfs:  cp.tmpfs,
		}
	case 4:
		//SIF
		sylog.Fatalf("Building from SIF not yet supported")
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

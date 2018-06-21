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
	"fmt"
	"io/ioutil"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/loop"
)

// LocalConveyor only needs to hold the conveyor to have the needed data to pack
type LocalConveyor struct {
	src string
}

// LocalConveyorPacker only needs to hold the conveyor to have the needed data to pack
type LocalConveyorPacker struct {
	LocalConveyor

	tmpfs string
}

// Get just stores the source
func (c *LocalConveyor) Get(recipe Definition) (err error) {
	c.src = recipe.Header["from"]
	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *LocalConveyorPacker) Pack() (b *Bundle, err error) {

	cp.tmpfs, err = ioutil.TempDir("", "temp-local-")
	if err != nil {
		return
	}

	b, err = NewBundle()

	// err = cp.unpackTmpfs(b)
	// if err != nil {
	// 	log.Fatal(err)
	// 	return
	// }

	var p Packer

	fmt.Println("Info Inside local packer", cp.src, cp.tmpfs)

	rootfs := cp.src

	//leverage C code to properly mount squashfs image
	C.singularity_config_init()

	imageObject := C.singularity_image_init(C.CString(rootfs), 0)

	info := new(loop.Info64)

	switch C.singularity_image_type(&imageObject) {
	case 1:
		//squashfs
		info.Offset = uint64(C.uint(imageObject.offset))
		info.SizeLimit = uint64(C.uint(imageObject.size))

		p = &SquashfsPacker{
			srcfile: rootfs,
			tmpfs:   cp.tmpfs,
			info:    info,
		}
	case 2:
		//ext3
		info.Offset = uint64(C.uint(imageObject.offset))
		info.SizeLimit = uint64(C.uint(imageObject.size))

		p = &Ext3Packer{
			srcfile: rootfs,
			tmpfs:   cp.tmpfs,
			info:    info,
		}
	default:
		sylog.Fatalf("Invalid image format from shub")
	}

	b, err = p.Pack()

	b.Recipe = Definition{}

	return b, nil
}

// func (cp *LocalConveyorPacker) unpackTmpfs(b *Bundle) (err error) {
// 	var p Packer

// 	fmt.Println("Info passed to unpackTmpfs", cp.src, cp.tmpfs)

// 	rootfs := cp.src

// 	//leverage C code to properly mount squashfs image
// 	C.singularity_config_init()

// 	imageObject := C.singularity_image_init(C.CString(rootfs), 0)

// 	info := new(loop.Info64)

// 	switch C.singularity_image_type(&imageObject) {
// 	case 1:
// 		//squashfs
// 		info.Offset = uint64(C.uint(imageObject.offset))
// 		info.SizeLimit = uint64(C.uint(imageObject.size))

// 		p = &SquashfsPacker{
// 			srcfile: rootfs,
// 			tmpfs:   cp.tmpfs,
// 			info:    info,
// 		}
// 	case 2:
// 		//ext3
// 		info.Offset = uint64(C.uint(imageObject.offset))
// 		info.SizeLimit = uint64(C.uint(imageObject.size))

// 		p = &Ext3Packer{
// 			srcfile: rootfs,
// 			tmpfs:   cp.tmpfs,
// 			info:    info,
// 		}
// 	default:
// 		sylog.Fatalf("Invalid image format from shub")
// 	}

// 	b, err = p.Pack()

// 	return err
// }

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/loop"
)

// SIFPacker holds the locations of where to pack from and to, aswell as image offset info
type SIFPacker struct {
	srcfile string
	tmpfs   string
	info    *loop.Info64
}

// Pack puts relevant objects in a Bundle!
func (p *SIFPacker) Pack() (b *Bundle, err error) {
	rootfs := p.srcfile

	b, err = NewBundle(p.tmpfs)
	if err != nil {
		return
	}
	err = p.unpackSIF(b, rootfs)
	if err != nil {
		sylog.Errorf("unpackSIF Failed", err.Error())
		return nil, err
	}

	return b, nil
}

// unpackSIF parses throught the sif file and places each component in the sandbox
func (p *SIFPacker) unpackSIF(b *Bundle, rootfs string) (err error) {

	//use sif tool(or its API whatever...) to look a the different sections
	//iterate over the sections and unpack them into the bundle, as a fs or as data

	return err
}

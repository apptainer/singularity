/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package build

import (
	//"fmt"
	//"io/ioutil"
	"io"
	"os"
	"path"
	//"time"

	"github.com/singularityware/singularity/pkg/image"
)

type SifBuilder struct {
	Definition
	Image *image.SIF
	path  string
	p     Provisioner
	tmpfs *image.Sandbox

	Out      io.Reader
	outWrite *os.File
	outRead  *os.File
	errWrite *os.File
	errRead  *os.File
}

func NewSifBuilder(imagePath string, d Definition) (b *SifBuilder, err error) {
	r, w, _ := os.Pipe()
	er, ew, _ := os.Pipe()

	b = &SifBuilder{
		Definition: d,
		path:       imagePath,
		Out:        io.MultiReader(r, er),
		outWrite:   w,
		errWrite:   ew,
		outRead:    r,
		errRead:    er,
	}

	b.tmpfs, err = image.TempSandbox(path.Base(imagePath) + "-")
	if err != nil {
		return b, err
	}

	uri := d.Header["bootstrap"] + "://" + d.Header["from"]
	b.p, err = NewProvisionerFromURI(uri)

	return b, err
}

func (b *SifBuilder) Build() {
	oldstdout := os.Stdout
	oldstderr := os.Stderr

	os.Stdout = b.outWrite
	os.Stderr = b.errWrite

	defer func() {
		os.Stdout = oldstdout
		os.Stderr = oldstderr
		b.outRead.Close()
		b.outWrite.Close()
		b.errRead.Close()
		b.errWrite.Close()
	}()

	b.p.Provision(b.tmpfs)
	img, err := image.SIFFromSandbox(b.tmpfs, b.path)
	if err != nil {
		return
	}

	b.Image = img
}

func (b *SifBuilder) createSifFile() (err error) {
	return nil
}

func (b *SifBuilder) mksquashfs() (err error) {
	return nil
}

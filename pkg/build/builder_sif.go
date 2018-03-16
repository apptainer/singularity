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

	Out      *os.File
	outWrite *os.File
}

func NewSifBuilder(imagePath string, d Definition) (b *SifBuilder, err error) {
	r, w, _ := os.Pipe()
	b = &SifBuilder{
		Definition: d,
		path:       imagePath,
		Out:        r,
		outWrite:   w,
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
	os.Stdout = b.outWrite

	defer func() {
		os.Stdout = oldstdout
		b.Out.Close()
		b.outWrite.Close()
	}()

	b.p.Provision(b.tmpfs)
	img, err := image.SIFFromSandbox(b.tmpfs, b.path)
	if err != nil {
		return
	}

	b.Image = img

	//b.Out.SetReadDeadline(time.Now().Add(time.Second))
	//out, _ := ioutil.ReadAll(b.Out)

	//b.outWrite.Close()
	//b.Out.Close()
	//os.Stdout = oldstdout

	//fmt.Println(string(out))
}

func (b *SifBuilder) createSifFile() (err error) {
	return nil
}

func (b *SifBuilder) mksquashfs() (err error) {
	return nil
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"context"
	"encoding/json"
	"io"
	"path"

	"github.com/singularityware/singularity/src/pkg/image"
	"github.com/singularityware/singularity/src/pkg/sif"
)

// SIFBuilder is an interface that enables building a SIF image.
type SIFBuilder struct {
	Def       Definition
	ImagePath string

	p     Provisioner
	image *sif.SIF
	tmpfs *image.Sandbox
}

// NewSIFBuilderJSON creates a new SIFBuilder using the supplied JSON.
func NewSIFBuilderJSON(imagePath string, r io.Reader) (b *SIFBuilder, err error) {
	var d Definition
	decoder := json.NewDecoder(r)

	for {
		if err = decoder.Decode(&d); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return NewSIFBuilder(imagePath, d)

	/*
		b.Def = d
		b.ImagePath = imagePath
		b.tmpfs, err = image.TempSandbox(path.Base(imagePath) + "-")
		if err != nil {
			return b, err
		}

		uri := d.Header["bootstrap"] + "://" + d.Header["from"]
		b.p, err = NewProvisionerFromURI(uri)

		return b, err*/
}

// NewSIFBuilder creates a new SIFBuilder from the supplied Definition struct
func NewSIFBuilder(imagePath string, d Definition) (b *SIFBuilder, err error) {
	b = &SIFBuilder{}

	b.Def = d
	b.ImagePath = imagePath
	b.tmpfs, err = image.TempSandbox(path.Base(imagePath) + "-")
	if err != nil {
		return b, err
	}

	uri := d.Header["bootstrap"] + "://" + d.Header["from"]
	b.p, err = NewProvisionerFromURI(uri)

	return b, err

}

// Build completes a build. The supplied context can be used for cancellation.
func (b *SIFBuilder) Build(ctx context.Context) (err error) {
	b.p.Provision(b.tmpfs)
	img, err := sif.FromSandbox(b.tmpfs, b.ImagePath)
	if err != nil {
		return err
	}

	b.image = img

	return nil
}

/*
func NewSifBuilder(imagePath string, d Definition) (b *SifBuilder, err error) {
	r, w, _ := os.Pipe()
	er, ew, _ := os.Pipe()

	b = &SifBuilder{
		Definition: d,
		path:       imagePath,
		Out:        io.MultiReader(er, r),
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
}*/

/*
// Build provisions a temporaly file system from a definition file
// and build a SIF afterwards.
func (b *SifBuilder) Build() (err error) {
	oldstdout := os.Stdout
	oldstderr := os.Stderr

	os.Stdout = b.outWrite
	os.Stderr = b.errWrite

	defer func() {
		os.Stdout = oldstdout
		os.Stderr = oldstderr
		//b.outRead.Close()
		b.outWrite.Close()
		//b.errRead.Close()
		b.errWrite.Close()
	}()

	err := b.p.Provision(b.tmpfs)
	if err != nil {
		return err
	}

	img, err := image.SIFFromSandbox(b.tmpfs, b.path)
	if err != nil {
		return err
	}

	b.Image = img

	return nil
}

func (b *SifBuilder) createSifFile() (err error) {
	return nil
}

func (b *SifBuilder) mksquashfs() (err error) {
	return nil
}
*/

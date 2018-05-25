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

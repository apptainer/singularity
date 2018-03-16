/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package build

import (
	"path"

	"github.com/singularityware/singularity/pkg/image"
)

type SifBuilder struct {
	Definition
	Image *image.SIF
	path  string
	p     Provisioner
	tmpfs *image.Sandbox
}

/*
func NewSifBuilderFromURI(imagePath string, uri string) (b *SifBuilder, err error) {
	u := strings.SplitN(uri, "://", 2)
	d := Definition{
		Header: map[string]string{
			"bootstrap": u[0],
			"from":      u[1],
		},
	}

	return NewSifBuilder(imagePath, d)
}
*/

func NewSifBuilder(imagePath string, d Definition) (b *SifBuilder, err error) {
	b = &SifBuilder{
		Definition: d,
		path:       imagePath,
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

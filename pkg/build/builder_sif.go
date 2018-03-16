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
	path string

	tmpfs *image.Sandbox
}

func NewSifBuilder(d Definition, p string) (b *SifBuilder, err error) {
	b = &SifBuilder{
		Definition: d,
	}

	b.tmpfs, err = image.TempSandbox(path.Base(p) + "-")
	if err != nil {
		return b, err
	}

	return b, nil
}

func (b *SifBuilder) Build() {

}

func (b *SifBuilder) createSifFile() (err error) {
	return nil
}

func (b *SifBuilder) mksquashfs() (err error) {
	return nil
}

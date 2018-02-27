/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package build

import (
	"github.com/singularityware/singularity/pkg/image"
)

// Incomplete or incorrect list of functions for now
type Builder interface {
	Build()
	/*Provisioner
	CreateMetadata()
	PreScript()
	SetupScript()
	PostScript()
	CopyFiles()*/
}
	
// Builder is
type Builder struct {
	Sandbox   image.Sandbox
	Image     image.Image
	BuildData buildData
	ImageData imageData
}


// createMetadataFolder installs /.singularity.d/* directory in the container.
// Serves as replacement of libexec/bootstrap-scripts/pre.sh
func (b Builder) createMetadataFolder() {

}

func (b Builder) PreScript() {

}

func (b Builder) PostScript() {

}

func (b Builder) SetupScript() {

}

func (b Builder) CopyFiles() {

}

func (b Builder) CopyEnvironment() {

}

func (b Builder) CopyRunscript() {

}

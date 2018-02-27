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

// Builder is
type Builder struct {
	Provisioner
	Sandbox   image.Sandbox
	Out       image.Image
	BuildData buildData
	ImageData imageData
}

type imageData struct {
	metadata []byte
	labels   []string
	imageScripts
}

type imageScripts struct {
	help        string
	environment string
	runscript   string
	test        string
}

type buildData struct {
	buildScripts
}

type buildScripts struct {
	pre   string
	setup string
	post  string
}

type metadata struct {
}

// createMetadataFolder installs /.singularity.d/* directory in the container.
// Serves as replacement of libexec/bootstrap-scripts/pre.sh
func (b *Builder) createMetadataFolder() {

}

func (b *Builder) PreScript() {

}

func (b *Builder) PostScript() {

}

func (b *Builder) SetupScript() {

}

func (b *Builder) CopyFiles() {

}

func (b *Builder) CopyEnvironment() {

}

func (b *Builder) CopyRunscript() {

}

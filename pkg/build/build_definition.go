/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package build

import ()

type BuildDefinition struct {
	ImageData imageData
	BuildData buildData
}

// imageData contains any scripts, metadata, etc... that needs to be
// present in some from in the final built image
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

// buildData contains any scripts, metadata, etc... that the Builder may
// need to know only at build time to build the image
type buildData struct {
	buildScripts
}

type buildScripts struct {
	pre   string
	setup string
	post  string
}

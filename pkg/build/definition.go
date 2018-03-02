/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package build

import ()

type Definition struct {
	Header    map[string]string
	ImageData imageData
	BuildData buildData
}

// imageData contains any scripts, metadata, etc... that needs to be
// present in some from in the final built image
type imageData struct {
	Metadata []byte   //
	Labels   []string //
	imageScripts
}

type imageScripts struct {
	Help        string
	Environment string
	Runscript   string
	Test        string
}

// buildData contains any scripts, metadata, etc... that the Builder may
// need to know only at build time to build the image
type buildData struct {
	Files map[string]string //
	buildScripts
}

type buildScripts struct {
	Pre   string
	Setup string
	Post  string
}

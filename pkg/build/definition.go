/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package build

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

// validSections just contains a list of all the valid sections a definition file
// could contain. If any others are found, an error will generate
var validSections = map[string]bool{
	"help":        true,
	"setup":       true,
	"files":       true,
	"labels":      true,
	"environment": true,
	"pre":         true,
	"post":        true,
	"runscript":   true,
	"test":        true,
}

// validHeaders just contains a list of all the valid headers a definition file
// could contain. If any others are found, an error will generate
var validHeaders = map[string]bool{
	"bootstrap":  true,
	"from":       true,
	"registry":   true,
	"namespace":  true,
	"includecmd": true,
	"mirrorurl":  true,
	"osversion":  true,
	"include":    true,
}

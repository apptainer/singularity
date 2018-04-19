/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package build

import "strings"

type Definition struct {
	Header map[string]string
	ImageData
	BuildData
}

// ImageData contains any scripts, metadata, etc... that needs to be
// present in some from in the final built image
type ImageData struct {
	Metadata []byte   //
	Labels   []string //
	ImageScripts
}

type ImageScripts struct {
	Help        string
	Environment string
	Runscript   string
	Test        string
}

// BuildData contains any scripts, metadata, etc... that the Builder may
// need to know only at build time to build the image
type BuildData struct {
	Files map[string]string //
	BuildScripts
}

type BuildScripts struct {
	Pre   string
	Setup string
	Post  string
}

func NewDefinitionFromURI(uri string) (d Definition, err error) {
	u := strings.SplitN(uri, "://", 2)

	d = Definition{
		Header: map[string]string{
			"bootstrap": u[0],
			"from":      u[1],
		},
	}

	return d, nil
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
	"includecmd": true,
	"mirrorurl":  true,
	"osversion":  true,
	"include":    true,
}

/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package build

import "strings"

type Definition struct {
	Header    map[string]string `json:"header"`
	ImageData `json:"imageData"`
	BuildData `json:"buildData"`
}

// ImageData contains any scripts, metadata, etc... that needs to be
// present in some from in the final built image
type ImageData struct {
	Metadata     []byte   `json:"metadata"`
	Labels       []string `json:"labels"`
	ImageScripts `json:"imageScripts"`
}

type ImageScripts struct {
	Help        string `json:"help"`
	Environment string `json:"environment"`
	Runscript   string `json:"runScript"`
	Test        string `json:"test"`
}

// BuildData contains any scripts, metadata, etc... that the Builder may
// need to know only at build time to build the image
type BuildData struct {
	Files        map[string]string `json:"files"`
	BuildScripts `json:"buildScripts"`
}

type BuildScripts struct {
	Pre   string `json:"pre"`
	Setup string `json:"setup"`
	Post  string `json:"post"`
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

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package types

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Definition describes how to build an image.
type Definition struct {
	Header    map[string]string `json:"header"`
	ImageData `json:"imageData"`
	BuildData Data `json:"buildData"`
}

// ImageData contains any scripts, metadata, etc... that needs to be
// present in some from in the final built image
type ImageData struct {
	Metadata     []byte            `json:"metadata"`
	Labels       map[string]string `json:"labels"`
	ImageScripts `json:"imageScripts"`
}

// ImageScripts contains scripts that are used after build time.
type ImageScripts struct {
	Help        string `json:"help"`
	Environment string `json:"environment"`
	Runscript   string `json:"runScript"`
	Test        string `json:"test"`
	Startscript string `json:"startScript"`
}

// Data contains any scripts, metadata, etc... that the Builder may
// need to know only at build time to build the image
type Data struct {
	Files   []FileTransport `json:"files"`
	Scripts `json:"buildScripts"`
}

// FileTransport holds source and destination information of files to copy into the container
type FileTransport struct {
	Src string `json:"source"`
	Dst string `json:"destination"`
}

// Scripts defines scripts that are used at build time.
type Scripts struct {
	Pre   string `json:"pre"`
	Setup string `json:"setup"`
	Post  string `json:"post"`
	Test  string `json:"test"`
}

// NewDefinitionFromURI crafts a new Definition given a URI
func NewDefinitionFromURI(uri string) (d Definition, err error) {
	var u []string
	if strings.Contains(uri, "://") {
		u = strings.SplitN(uri, "://", 2)
	} else if strings.Contains(uri, ":") {
		u = strings.SplitN(uri, ":", 2)
	} else {
		return d, fmt.Errorf("build URI must start with prefix:// or prefix: ")
	}

	d = Definition{
		Header: map[string]string{
			"bootstrap": u[0],
			"from":      u[1],
		},
	}

	return d, nil
}

// NewDefinitionFromJSON creates a new Definition using the supplied JSON.
func NewDefinitionFromJSON(r io.Reader) (d Definition, err error) {
	decoder := json.NewDecoder(r)

	for {
		if err = decoder.Decode(&d); err == io.EOF {
			break
		} else if err != nil {
			return
		}
	}

	return d, nil
}

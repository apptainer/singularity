// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

// Definition describes how to build an image.
type Definition struct {
	Header     map[string]string `json:"header"`
	ImageData  `json:"imageData"`
	BuildData  Data              `json:"buildData"`
	CustomData map[string]string `json:"customData"`
	Raw        []byte            `json:"raw"`
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

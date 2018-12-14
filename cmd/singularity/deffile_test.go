// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"path"
)

type DefFileDetail struct {
	Bootstrap string
	From      string
	Registry  string
	Namespace string
	Labels    map[string]string
}

// prepareTemplate reads a template from a file, applies data to it, writes the
// contents to disk, and returns the path.
func prepareDefFile(dfd DefFileDetail) (outputPath string) {
	tmpl, err := template.ParseFiles(path.Join("testdata", "deffile.tmpl"))
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	f, err := ioutil.TempFile("", "TestTemplate-")
	if err != nil {
		log.Fatalf("failed to open temp file: %v", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, dfd); err != nil {
		log.Fatalf("failed to execute template: %v", err)
	}

	return f.Name()
}

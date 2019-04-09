// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"log"
	"path"
)

type DefFileDetail struct {
	Bootstrap   string
	From        string
	Registry    string
	Namespace   string
	Stage       string
	Help        []string
	Env         []string
	Labels      map[string]string
	Files       []FilePair
	FilesFrom   []FileSection
	Pre         []string
	Setup       []string
	Post        []string
	RunScript   []string
	Test        []string
	StartScript []string
	Apps        []AppDetail
}

type AppDetail struct {
	Name    string
	Help    []string
	Env     []string
	Labels  map[string]string
	Files   []FilePair
	Install []string
	Run     []string
	Test    []string
}

type FileSection struct {
	Stage string
	Files []FilePair
}

type FilePair struct {
	Src string
	Dst string
}

// prepareDefFile reads a template from a file, applies data to it, writes the
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

// prepareDefFile reads a template from a file, applies data to it for each definition,
// concatenates them all together, writes them to a file and returns the path.
func prepareMultipleDefFiles(dfd []DefFileDetail) (outputPath string) {
	var b bytes.Buffer
	for _, d := range dfd {
		tmpl, err := template.ParseFiles(path.Join("testdata", "deffile.tmpl"))
		if err != nil {
			log.Fatalf("failed to parse template: %v", err)
		}

		if err := tmpl.Execute(&b, d); err != nil {
			log.Fatalf("failed to execute template: %v", err)
		}
	}

	f, err := ioutil.TempFile("", "TestTemplate-")
	if err != nil {
		log.Fatalf("failed to open temp file: %v", err)
	}
	defer f.Close()

	if _, err := f.Write(b.Bytes()); err != nil {
		log.Fatalf("failed to write temp file: %v", err)
	}

	return f.Name()
}

// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package types

import (
	"os"
	"strings"
	"testing"
)

func TestNewDefinitionFromURI(t *testing.T) {
	invalidURIs := []string{"test", "//test"}
	validURIs := []string{"://test", ":test"}

	for _, invalidURI := range invalidURIs {
		_, myerr := NewDefinitionFromURI(invalidURI)
		if myerr == nil {
			t.Fatal("NewDefinitionFromURI() succeeded with an invalid URI:", invalidURI)
		}
	}

	for _, validURI := range validURIs {
		_, myerr := NewDefinitionFromURI(validURI)
		if myerr != nil {
			t.Fatal("NewDefinitionFromURI() failed with a valid URI:", validURI)
		}
	}
}

func TestNewDefinitionFromJSON(t *testing.T) {
	const invalidJSON = `{"test"}`
	const validJSON = `{"Key1": "Value1", "Key2": "Value2."}`
	const singularityJSON = "parser/testdata_good/docker/docker.json"
	// We do not have a valid example file that we can use to reach the corner cases, so we define a fake JSON
	const validSingularityJSON = `{"header":{"bootstrap":"yum","include":"yum","mirrorurl":"http://mirror.centos.org/centos-%{OSVERSION}/%{OSVERSION}/os/$basearch/","osversion":"7"},"imageData":{"metadata":null,"labels":{"Maintainer":"gvallee"},"imageScripts":{"help":"","environment":"","runScript":"","test":"testMyTest","startScript":""}},"buildData":{"files":[{"source":"myFakeFile"}],"buildScripts":{"pre":"","setup":"","post":"","test":""}},"customData":null}`

	_, myerr := NewDefinitionFromJSON(strings.NewReader(invalidJSON))
	if myerr == nil {
		t.Fatal("NewDefinitionFromJSON() succeeded with an invalid JSON")
	}

	_, myerr = NewDefinitionFromJSON(strings.NewReader(validJSON))
	if myerr != nil {
		t.Fatal("NewDefinitionFromJSON() failed with a valid JSON")
	}

	// Testing with a valid JSON file
	f, err := os.Open(singularityJSON)
	if err != nil {
		t.Fatal("cannot open test file", err)
	}
	var def1 Definition
	def1, myerr = NewDefinitionFromJSON(f)
	if myerr != nil {
		t.Fatal("NewDefinitionFromJSON() failed with a valid JSON")
	}
	if len(def1.ImageData.Labels) != 2 {
		t.Fatal("Invalid number of labels")
	}

	// Testing with a valid JSON with raw section
	var def2 Definition
	def2, myerr = NewDefinitionFromJSON(strings.NewReader(validSingularityJSON))
	if myerr != nil {
		t.Fatal("NewDefinitionFromJSON() failed with a Singularity JSON")
	}
	if len(def2.ImageData.Labels) != 1 {
		t.Fatal("Invalid number of labels")
	}
}

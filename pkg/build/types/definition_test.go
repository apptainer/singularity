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
	cases := []struct {
		uri        string
		shouldPass bool
	}{
		{uri: "test", shouldPass: false},
		{uri: "//test", shouldPass: false},
		{uri: "://test", shouldPass: true},
		{uri: ":test", shouldPass: true}}

	for _, testCase := range cases {
		_, myerr := NewDefinitionFromURI(testCase.uri)
		if testCase.shouldPass == false && myerr == nil {
			t.Fatal("NewDefinitionFromURI() succeeded with an invalid URI:", testCase.uri)
		}
		if testCase.shouldPass == true && myerr != nil {
			t.Fatal("NewDefinitionFromURI() failed with a valid URI:", myerr)
		}
	}
}

func TestNewDefinitionFromJSON(t *testing.T) {
	simpleCases := []struct {
		JSON       string
		shouldPass bool
	}{
		{JSON: `{"test"}`, shouldPass: false},
		{JSON: `{"Key1": "Value1", "Key2": "Value2."}`, shouldPass: true}}

	const singularityJSON = "parser/testdata_good/docker/docker.json"
	// We do not have a valid example file that we can use to reach the corner cases, so we define a fake JSON
	const validSingularityJSON = `{"header":{"bootstrap":"yum","include":"yum","mirrorurl":"http://mirror.centos.org/centos-%{OSVERSION}/%{OSVERSION}/os/$basearch/","osversion":"7"},"imageData":{"metadata":null,"labels":{"Maintainer":"gvallee"},"imageScripts":{"help":"","environment":"","runScript":"","test":"testMyTest","startScript":""}},"buildData":{"files":[{"source":"myFakeFile"}],"buildScripts":{"pre":"","setup":"","post":"","test":""}},"customData":null}`

	for _, testCase := range simpleCases {
		_, myerr := NewDefinitionFromJSON(strings.NewReader(testCase.JSON))
		if testCase.shouldPass == false && myerr == nil {
			t.Fatal("NewDefinitionFromJSON() succeeded with an invalid JSON")
		}
		if testCase.shouldPass == true && myerr != nil {
			t.Fatal("NewDefinitionFromJSON() failed with a valid JSON", myerr)
		}
	}

	// Testing with a valid JSON file
	f, err := os.Open(singularityJSON)
	if err != nil {
		t.Fatal("cannot open test file", err)
	}
	var def1 Definition
	def1, def1Err := NewDefinitionFromJSON(f)
	if def1Err != nil {
		t.Fatal("NewDefinitionFromJSON() failed with a valid JSON")
	}
	if len(def1.ImageData.Labels) != 2 {
		t.Fatal("Invalid number of labels")
	}

	// Testing with a valid JSON with raw section
	var def2 Definition
	def2, def2Err := NewDefinitionFromJSON(strings.NewReader(validSingularityJSON))
	if def2Err != nil {
		t.Fatal("NewDefinitionFromJSON() failed with a Singularity JSON")
	}
	if len(def2.ImageData.Labels) != 1 {
		t.Fatal("Invalid number of labels")
	}
}

// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package parser

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/pkg/build/types"
)

func TestScanDefinitionFile(t *testing.T) {
	tests := []struct {
		name     string
		defPath  string
		sections string
	}{
		{"Arch", "testdata_good/arch/arch", "testdata_good/arch/arch_sections.json"},
		{"BusyBox", "testdata_good/busybox/busybox", "testdata_good/busybox/busybox_sections.json"},
		{"Debootstrap", "testdata_good/debootstrap/debootstrap", "testdata_good/debootstrap/debootstrap_sections.json"},
		{"Docker", "testdata_good/docker/docker", "testdata_good/docker/docker_sections.json"},
		{"LocalImage", "testdata_good/localimage/localimage", "testdata_good/localimage/localimage_sections.json"},
		{"Scratch", "testdata_good/scratch/scratch", "testdata_good/scratch/scratch_sections.json"},
		{"Shub", "testdata_good/shub/shub", "testdata_good/shub/shub_sections.json"},
		{"Yum", "testdata_good/yum/yum", "testdata_good/yum/yum_sections.json"},
		{"Zypper", "testdata_good/zypper/zypper", "testdata_good/zypper/zypper_sections.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			deffile := tt.defPath
			r, err := os.Open(deffile)
			if err != nil {
				t.Fatal("failed to read deffile:", err)
			}
			defer r.Close()

			s := bufio.NewScanner(r)
			s.Split(scanDefinitionFile)
			for s.Scan() && s.Text() == "" && s.Err() == nil {
			}

			b, err := ioutil.ReadFile(tt.sections)
			if err != nil {
				t.Fatal("failed to read JSON:", err)
			}

			type DefFileSections struct {
				Header string
			}
			var d []DefFileSections
			if err := json.Unmarshal(b, &d); err != nil {
				t.Fatal("failed to unmarshal JSON:", err)
			}

			// Right now this only does the header, but the json files are
			// written with all of the sections in mind so that could be added.
			if s.Text() != d[0].Header {
				t.Fatal("scanDefinitionFile does not produce same header as reference")
			}

		}))
	}
}

// Specific test to cover some corner cases of splitToken()
//func TestSplitToken(t *testing.T) {
//	ident_str := "test test1"
//	content_str := "content1 content2"
//	str := "%%%%" + ident_str + "\n" + content_str
//	ident, content := splitToken(str)
//	if ident != ident_str || content != content_str {
//		t.Fatal("splitToken returned bad values")
//	}
//
//	str = "%%" + ident_str
//	ident, content = splitToken(str)
//	if ident != ident_str || content != "" {
//		t.Fatal("splitToken returned bad values")
//	}
//}

// Specific tests to cover some corner cases of parseTokenSection()
func TestParseTokenSection(t *testing.T) {
	// Fake map
	testMap := make(map[string]string)
	testMap["fakeKey1"] = "%content1 content2 content3"
	testMap["fakeKey2"] = ""

	// Incorrect token; map not used
	str := "test test1"
	myerr := parseTokenSection(str, nil)
	if myerr == nil {
		t.Fatal("test expected to fail but succeeded")
	}

	// Another incorrect token case; map not used
	myerr = parseTokenSection("apptest\ntest", nil)
	if myerr == nil {
		t.Fatal("test expected to fail but succeeded")
	}

	// Correct token
	myerr = parseTokenSection("appenv apptest apptest2\ntest", testMap)
	if myerr != nil {
		t.Fatal("error while parsing sections")
	}
	if testMap["appenv apptest"] != "test" {
		t.Fatal("returned map is invalid", testMap["appenv"])
	}
}

// Specific tests to cover some corner cases of doSections()
func TestDoSections(t *testing.T) {
	// This is an string representing an invalid section, we make sure it is not identified as a header
	invalid_str := "%apptest\ntesttext"

	// This is a fake data structure
	myData := new(types.Definition)
	myData.Labels = make(map[string]string)

	s1 := bufio.NewScanner(strings.NewReader(invalid_str))
	s1.Split(scanDefinitionFile)

	// advance scanner until it returns a useful token
	for s1.Scan() && s1.Text() == "" {
		// Nothing to do
	}

	myerr := doSections(s1, myData)
	if myerr == nil {
		t.Fatal("Test passed while expected to fail")
	}

	// Now we define a valid first section but an invalid second section
	invalid_str = "%appenv apptest apptest2\ntest\n%appenv\ntest"
	s2 := bufio.NewScanner(strings.NewReader(invalid_str))
	s2.Split(scanDefinitionFile)

	// Advance the scanner until it returns a useful token
	for s2.Scan() && s2.Text() == "" {
		// Nothing to do
	}

	myerr = doSections(s2, myData)
	if myerr == nil {
		t.Fatal("Test passed while expected to fail")
	}
}

func TestParseDefinitionFile(t *testing.T) {
	tests := []struct {
		name     string
		defPath  string
		jsonPath string
	}{
		{"Arch", "testdata_good/arch/arch", "testdata_good/arch/arch.json"},
		{"BusyBox", "testdata_good/busybox/busybox", "testdata_good/busybox/busybox.json"},
		{"Debootstrap", "testdata_good/debootstrap/debootstrap", "testdata_good/debootstrap/debootstrap.json"},
		{"Docker", "testdata_good/docker/docker", "testdata_good/docker/docker.json"},
		{"LocalImage", "testdata_good/localimage/localimage", "testdata_good/localimage/localimage.json"},
		{"Scratch", "testdata_good/scratch/scratch", "testdata_good/scratch/scratch.json"},
		{"Shub", "testdata_good/shub/shub", "testdata_good/shub/shub.json"},
		{"Yum", "testdata_good/yum/yum", "testdata_good/yum/yum.json"},
		{"Zypper", "testdata_good/zypper/zypper", "testdata_good/zypper/zypper.json"},
		{"NoHeader", "testdata_good/noheader/noheader", "testdata_good/noheader/noheader.json"},
		{"NoHeaderComments", "testdata_good/noheadercomments/noheadercomments", "testdata_good/noheadercomments/noheadercomments.json"},
		{"NoHeaderWhiteSpace", "testdata_good/noheaderwhitespace/noheaderwhitespace", "testdata_good/noheaderwhitespace/noheaderwhitespace.json"},
		{"MultipleScripts", "testdata_good/multiplescripts/multiplescripts", "testdata_good/multiplescripts/multiplescripts.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			defFile, err := os.Open(tt.defPath)
			if err != nil {
				t.Fatal("failed to open:", err)
			}
			defer defFile.Close()

			jsonFile, err := os.OpenFile(tt.jsonPath, os.O_RDWR, 0755)
			if err != nil {
				t.Fatal("failed to open:", err)
			}
			defer jsonFile.Close()

			defTest, err := ParseDefinitionFile(defFile)
			if err != nil {
				t.Fatal("failed to parse definition file:", err)
			}

			var defCorrect types.Definition
			if err := json.NewDecoder(jsonFile).Decode(&defCorrect); err != nil {
				t.Fatal("failed to parse JSON:", err)
			}

			if !reflect.DeepEqual(defTest, defCorrect) {
				t.Fatal("parsed definition did not match reference")
			}
		}))
	}
}

func TestParseDefinitionFileFailure(t *testing.T) {
	tests := []struct {
		name    string
		defPath string
	}{
		{"BadSection", "testdata_bad/bad_section"},
		{"JSONInput1", "testdata_bad/json_input_1"},
		{"JSONInput2", "testdata_bad/json_input_2"},
		{"Empty", "testdata_bad/empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			defFile, err := os.Open(tt.defPath)
			if err != nil {
				t.Fatal("failed to open:", err)
			}
			defer defFile.Close()

			if _, err = ParseDefinitionFile(defFile); err == nil {
				t.Fatal("unexpected success parsing definition file")
			}
		}))
	}
}

// Specific tests to cover some corner cases of IsInvalidSectionError()
func TestIsInvalidSectionErrors(t *testing.T) {

	// Test of IsInvalidSectionError()
	dummy_keys := []string{"dummy_key1", "dummy_key2"}
	myValidErr1 := &InvalidSectionError{dummy_keys, errInvalidSection}
	myValidErr2 := &InvalidSectionError{dummy_keys, errEmptyDefinition}
	myInvalidErr := errors.New("My dummy error")
	if IsInvalidSectionError(myValidErr1) == false ||
		IsInvalidSectionError(myValidErr2) == false ||
		IsInvalidSectionError(myInvalidErr) == true {
		t.Fatal("unexpecter return value for IsInvalidSectionError()")
	}

	// Test of Error()
	expectedStr1 := "invalid section(s) specified: " + strings.Join(dummy_keys, ", ")
	expectedStr2 := "Empty definition file: " + strings.Join(dummy_keys, ", ")
	if myValidErr1.Error() != expectedStr1 || myValidErr2.Error() != expectedStr2 {
		t.Fatal("unexpecter result from Error()", myValidErr1.Error())
	}
}

// Specific tests to cover some corner cases of PopulateDefinition()
func TestPopulateDefinition(t *testing.T) {
	//
	// Some variables used throughout the tests
	//

	// We use a specific set of section names to reach some corner cases
	testMap := make(map[string]string)
	testMap["files"] = "file1 file2"
	testMap["labels"] = "label1"

	emptyMap := make(map[string]string)

	//
	// Test with invalid data
	//
	invalidData := new(types.Definition)
	invalidData.Labels = make(map[string]string)
	populateDefinition(emptyMap, invalidData)

	//
	// Test with very specific maps
	//

	// A structure to store results (not really relevant here)
	myData := new(types.Definition)
	myData.Labels = make(map[string]string)

	myerr := populateDefinition(testMap, myData)
	if myerr != nil {
		t.Fatal("Test failed while testing populateDefinition()")
	}
}

// Specific tests to cover some corners cases of doHeader()
func TestDoHeader(t *testing.T) {
	invalidHeader := "headerTest"
	myData := new(types.Definition)
	myData.Labels = make(map[string]string)

	myerr := doHeader(invalidHeader, myData)
	if myerr == nil {
		t.Fatal("Test succeeded while supposed to fail")
	}
}

func TestIsValidDefinition(t *testing.T) {

	//
	// Test with a bunch of valid files
	//
	valid_tests := []struct {
		name     string
		defPath  string
		sections string
	}{
		{"Arch", "testdata_good/arch/arch", "testdata_good/arch/arch_sections.json"},
		{"BusyBox", "testdata_good/busybox/busybox", "testdata_good/busybox/busybox_sections.json"},
		{"Debootstrap", "testdata_good/debootstrap/debootstrap", "testdata_good/debootstrap/debootstrap_sections.json"},
		{"Docker", "testdata_good/docker/docker", "testdata_good/docker/docker_sections.json"},
		{"LocalImage", "testdata_good/localimage/localimage", "testdata_good/localimage/localimage_sections.json"},
		{"Scratch", "testdata_good/scratch/scratch", "testdata_good/scratch/scratch_sections.json"},
		{"Shub", "testdata_good/shub/shub", "testdata_good/shub/shub_sections.json"},
		{"Yum", "testdata_good/yum/yum", "testdata_good/yum/yum_sections.json"},
		{"Zypper", "testdata_good/zypper/zypper", "testdata_good/zypper/zypper_sections.json"},
	}

	for _, tt := range valid_tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			deffile := tt.defPath

			valid, err := IsValidDefinition(deffile)
			if valid == false || err != nil {
				t.Fatal("Validation of a definition file failed while expected to succeed")
			}
		}))
	}

	//
	// Test with a non-existing file
	//
	valid, err := IsValidDefinition("notExistingDirectory/notExistingFile")
	if valid == true && err != nil {
		t.Fatal("Validation of a non-existing file succeeded while expected to fail")
	}

	//
	// Test passing a valid directory in instead of a file
	//
	valid, err = IsValidDefinition("testdata_bad")
	if valid == true && err != nil {
		t.Fatal("Validation of a directory succeeded while expected to fail")
	}

	//
	// Now test with invalid definition files
	//
	invalid_tests := []struct {
		name    string
		defPath string
	}{
		{"BadSection", "testdata_bad/bad_section"},
		{"JSONInput1", "testdata_bad/json_input_1"},
		{"JSONInput2", "testdata_bad/json_input_2"},
		{"Empty", "testdata_bad/empty"},
	}
	for _, tt := range invalid_tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			deffile := tt.defPath

			valid, err := IsValidDefinition(deffile)
			if valid == true && err == nil {
				t.Fatal("Validation of an invalid definition file succeeded while expected to fail")
			}
		}))
	}

}

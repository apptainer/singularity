// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package parser

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
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
		{"Arch", "../../../../internal/pkg/build/testdata_good/arch/arch", "../../../../internal/pkg/build/testdata_good/arch/arch_sections.json"},
		{"BusyBox", "../../../../internal/pkg/build/testdata_good/busybox/busybox", "../../../../internal/pkg/build/testdata_good/busybox/busybox_sections.json"},
		{"Debootstrap", "../../../../internal/pkg/build/testdata_good/debootstrap/debootstrap", "../../../../internal/pkg/build/testdata_good/debootstrap/debootstrap_sections.json"},
		{"Docker", "../../../../internal/pkg/build/testdata_good/docker/docker", "../../../../internal/pkg/build/testdata_good/docker/docker_sections.json"},
		{"LocalImage", "../../../../internal/pkg/build/testdata_good/localimage/localimage", "../../../../internal/pkg/build/testdata_good/localimage/localimage_sections.json"},
		{"Shub", "../../../../internal/pkg/build/testdata_good/shub/shub", "../../../../internal/pkg/build/testdata_good/shub/shub_sections.json"},
		{"Yum", "../../../../internal/pkg/build/testdata_good/yum/yum", "../../../../internal/pkg/build/testdata_good/yum/yum_sections.json"},
		{"Zypper", "../../../../internal/pkg/build/testdata_good/zypper/zypper", "../../../../internal/pkg/build/testdata_good/zypper/zypper_sections.json"},
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

func TestParseDefinitionFile(t *testing.T) {
	tests := []struct {
		name     string
		defPath  string
		jsonPath string
	}{
		{"Arch", "../../../../internal/pkg/build/testdata_good/arch/arch", "../../../../internal/pkg/build/testdata_good/arch/arch.json"},
		{"BusyBox", "../../../../internal/pkg/build/testdata_good/busybox/busybox", "../../../../internal/pkg/build/testdata_good/busybox/busybox.json"},
		{"Debootstrap", "../../../../internal/pkg/build/testdata_good/debootstrap/debootstrap", "../../../../internal/pkg/build/testdata_good/debootstrap/debootstrap.json"},
		{"Docker", "../../../../internal/pkg/build/testdata_good/docker/docker", "../../../../internal/pkg/build/testdata_good/docker/docker.json"},
		{"LocalImage", "../../../../internal/pkg/build/testdata_good/localimage/localimage", "../../../../internal/pkg/build/testdata_good/localimage/localimage.json"},
		{"Shub", "../../../../internal/pkg/build/testdata_good/shub/shub", "../../../../internal/pkg/build/testdata_good/shub/shub.json"},
		{"Yum", "../../../../internal/pkg/build/testdata_good/yum/yum", "../../../../internal/pkg/build/testdata_good/yum/yum.json"},
		{"Zypper", "../../../../internal/pkg/build/testdata_good/zypper/zypper", "../../../../internal/pkg/build/testdata_good/zypper/zypper.json"},
		{"NoHeader", "../../../../internal/pkg/build/testdata_good/noheader/noheader", "../../../../internal/pkg/build/testdata_good/noheader/noheader.json"},
		{"NoHeaderComments", "../../../../internal/pkg/build/testdata_good/noheadercomments/noheadercomments", "../../../../internal/pkg/build/testdata_good/noheadercomments/noheadercomments.json"},
		{"NoHeaderWhiteSpace", "../../../../internal/pkg/build/testdata_good/noheaderwhitespace/noheaderwhitespace", "../../../../internal/pkg/build/testdata_good/noheaderwhitespace/noheaderwhitespace.json"},
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
		{"BadSection", "../../../../internal/pkg/build/testdata_bad/bad_section"},
		{"JSONInput1", "../../../../internal/pkg/build/testdata_bad/json_input_1"},
		{"JSONInput2", "../../../../internal/pkg/build/testdata_bad/json_input_2"},
		{"Empty", "../../../../internal/pkg/build/testdata_bad/empty"},
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

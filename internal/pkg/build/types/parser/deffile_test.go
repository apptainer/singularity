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

	"github.com/sylabs/singularity/internal/pkg/build/types"
	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestScanDefinitionFile(t *testing.T) {
	tests := []struct {
		name     string
		defPath  string
		sections string
	}{
		{"Arch", "../../testdata_good/arch/arch", "../../testdata_good/arch/arch_sections.json"},
		{"BusyBox", "../../testdata_good/busybox/busybox", "../../testdata_good/busybox/busybox_sections.json"},
		{"Debootstrap", "../../testdata_good/debootstrap/debootstrap", "../../testdata_good/debootstrap/debootstrap_sections.json"},
		{"Docker", "../../testdata_good/docker/docker", "../../testdata_good/docker/docker_sections.json"},
		{"LocalImage", "../../testdata_good/localimage/localimage", "../../testdata_good/localimage/localimage_sections.json"},
		{"Scratch", "../../testdata_good/scratch/scratch", "../../testdata_good/scratch/scratch_sections.json"},
		{"Shub", "../../testdata_good/shub/shub", "../../testdata_good/shub/shub_sections.json"},
		{"Yum", "../../testdata_good/yum/yum", "../../testdata_good/yum/yum_sections.json"},
		{"Zypper", "../../testdata_good/zypper/zypper", "../../testdata_good/zypper/zypper_sections.json"},
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
		{"Arch", "../../testdata_good/arch/arch", "../../testdata_good/arch/arch.json"},
		{"BusyBox", "../../testdata_good/busybox/busybox", "../../testdata_good/busybox/busybox.json"},
		{"Debootstrap", "../../testdata_good/debootstrap/debootstrap", "../../testdata_good/debootstrap/debootstrap.json"},
		{"Docker", "../../testdata_good/docker/docker", "../../testdata_good/docker/docker.json"},
		{"LocalImage", "../../testdata_good/localimage/localimage", "../../testdata_good/localimage/localimage.json"},
		{"Scratch", "../../testdata_good/scratch/scratch", "../../testdata_good/scratch/scratch.json"},
		{"Shub", "../../testdata_good/shub/shub", "../../testdata_good/shub/shub.json"},
		{"Yum", "../../testdata_good/yum/yum", "../../testdata_good/yum/yum.json"},
		{"Zypper", "../../testdata_good/zypper/zypper", "../../testdata_good/zypper/zypper.json"},
		{"NoHeader", "../../testdata_good/noheader/noheader", "../../testdata_good/noheader/noheader.json"},
		{"NoHeaderComments", "../../testdata_good/noheadercomments/noheadercomments", "../../testdata_good/noheadercomments/noheadercomments.json"},
		{"NoHeaderWhiteSpace", "../../testdata_good/noheaderwhitespace/noheaderwhitespace", "../../testdata_good/noheaderwhitespace/noheaderwhitespace.json"},
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
		//{"BadSection", "../../testdata_bad/bad_section"},
		{"JSONInput1", "../../testdata_bad/json_input_1"},
		{"JSONInput2", "../../testdata_bad/json_input_2"},
		{"Empty", "../../testdata_bad/empty"},
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

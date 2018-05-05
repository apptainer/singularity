/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package build

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestParseDefinitionFile(t *testing.T) {
	tests := []struct {
		name     string
		defPath  string
		jsonPath string
	}{
		{"Docker", "./testdata_good/docker/docker", "./testdata_good/docker/docker.json"},
		{"BusyBox", "./testdata_good/busybox/busybox", "./testdata_good/busybox/busybox.json"},
		{"Debootstrap", "./testdata_good/debootstrap/debootstrap", "./testdata_good/debootstrap/debootstrap.json"},
		{"Arch", "./testdata_good/arch/arch", "./testdata_good/arch/arch.json"},
		{"LocalImage", "./testdata_good/localimage/localimage", "./testdata_good/localimage/localimage.json"},
		{"Shub", "./testdata_good/shub/shub", "./testdata_good/shub/shub.json"},
		{"Yum", "./testdata_good/yum/yum", "./testdata_good/yum/yum.json"},
		{"Zypper", "./testdata_good/zypper/zypper", "./testdata_good/zypper/zypper.json"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defFile, err := os.Open(test.defPath)
			if err != nil {
				t.Fatal("failed to open:", err)
			}
			defer defFile.Close()

			jsonFile, err := os.OpenFile(test.jsonPath, os.O_RDWR, 0755)
			if err != nil {
				t.Fatal("failed to open:", err)
			}
			defer jsonFile.Close()

			defTest, err := ParseDefinitionFile(defFile)
			if err != nil {
				t.Fatal("failed to parse definition file:", err)
			}

			// json.NewEncoder(jsonFile).Encode(&defTest)

			var defCorrect Definition
			if err := json.NewDecoder(jsonFile).Decode(&defCorrect); err != nil {
				t.Fatal("failed to parse JSON:", err)
			}

			if !reflect.DeepEqual(defTest, defCorrect) {
				t.Log(defTest)
				t.Log(defCorrect)
				t.Fatal("parsed definition did not match reference")
			}
		})
	}
}

func TestParseDefinitionFileFailure(t *testing.T) {
	tests := []struct {
		name    string
		defPath string
	}{
		{"BadSection", "./testdata_bad/bad_section"},
		{"JSONInput1", "./testdata_bad/json_input_1"},
		{"JSONInput2", "./testdata_bad/json_input_2"},
		{"Empty", "./testdata_bad/empty"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defFile, err := os.Open(test.defPath)
			if err != nil {
				t.Fatal("failed to open:", err)
			}
			defer defFile.Close()

			if _, err = ParseDefinitionFile(defFile); err == nil {
				t.Fatal("unexpected success parsing definition file", err)
			}
		})
	}
}

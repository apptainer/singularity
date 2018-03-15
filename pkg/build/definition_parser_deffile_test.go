/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package build

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"
)

func helperParseAndCompare(t *testing.T, defPath string, jsonPath string) (err error) {
	defFile, err := os.Open(defPath)
	defer defFile.Close()
	if err != nil {
		t.Error(err)
		return
	}

	jsonFile, err := os.Open(jsonPath)
	defer jsonFile.Close()
	if err != nil {
		t.Error(err)
		return
	}

	defTest, err := ParseDefinitionFile(defFile)
	if err != nil {
		t.Error(err)
		return
	}

	d := json.NewDecoder(jsonFile)
	for {
		var defCorrect Definition
		if err := d.Decode(&defCorrect); err == io.EOF {
			break
		} else if err != nil {
			t.Error(err)
			return err
		}

		if !reflect.DeepEqual(defTest, defCorrect) {
			return fmt.Errorf("Failed to correctly parse definition file: %s", defPath)
		}
	}

	return nil
}

func helperParseBad(t *testing.T, defPath string) (err error) {
	defFile, err := os.Open(defPath)
	defer defFile.Close()
	if err != nil {
		t.Error(err)
		return
	}

	_, err = ParseDefinitionFile(defFile)
	if err != nil {
		return nil
	} else {
		return fmt.Errorf("ParseDefinitionFile incorrectly succeeded in parsing file: %s", defPath)
	}
}

func TestParseDefinitionFile(t *testing.T) {
	// Map[path]path.json
	definitionFilesGood := map[string]string{
		"./testdata_good/docker/docker":           "./testdata_good/docker/docker.json",
		"./testdata_good/busybox/busybox":         "./testdata_good/busybox/busybox.json",
		"./testdata_good/debootstrap/debootstrap": "./testdata_good/debootstrap/debootstrap.json",
		"./testdata_good/arch/arch":               "./testdata_good/arch/arch.json",
		"./testdata_good/localimage/localimage":   "./testdata_good/localimage/localimage.json",
		"./testdata_good/shub/shub":               "./testdata_good/shub/shub.json",
		"./testdata_good/yum/yum":                 "./testdata_good/yum/yum.json",
		"./testdata_good/zypper/zypper":           "./testdata_good/zypper/zypper.json",
	}

	definitionFilesBad := []string{
		"./testdata_bad/bad_section",
		"./testdata_bad/json_input_1",
		"./testdata_bad/json_input_2",
	}

	for defPath, jsonPath := range definitionFilesGood {
		err := helperParseAndCompare(t, defPath, jsonPath)
		if err != nil {
			t.Error(err)
		}
	}

	for _, defPath := range definitionFilesBad {
		err := helperParseBad(t, defPath)
		if err != nil {
			t.Error(err)
		}

	}
}

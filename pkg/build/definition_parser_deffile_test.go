/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package build

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestParseDefinitionFile(t *testing.T) {
	testFilesOK := map[string]string{
		"docker":      "./mock/docker/docker",
		"debootstrap": "./mock/debootstrap/debootstrap",
		"arch":        "./mock/arch/arch",
		"yum":         "./mock/yum/yum",
		"shub":        "./mock/shub/shub",
		"localimage":  "./mock/localimage/localimage",
		"busybox":     "./mock/busybox/busybox",
		"zypper":      "./mock/zypper/zypper",
	}
	testFilesBAD := map[string]string{
		"bad_section": "./mock/bad_section/bad_section",
	}
	resultFile := map[string]string{
		"docker":      "./mock/docker/result",
		"debootstrap": "./mock/debootstrap/result",
		"arch":        "./mock/arch/result",
		"yum":         "./mock/yum/result",
		"shub":        "./mock/shub/result",
		"localimage":  "./mock/localimage/result",
		"busybox":     "./mock/busybox/result",
		"zypper":      "./mock/zypper/result",
	}

	// Loop through the Deffiles OK
	for k := range testFilesOK {
		t.Logf("=>\tRunning test for Deffile:\t\t[%s]", k)
		f, err := ioutil.TempFile(os.TempDir(), fmt.Sprintf("singularity_parser_test_%s", k))
		if err != nil {
			t.Log(err)
			t.Fail()
		}
		defer os.Remove(f.Name())

		r, err := os.Open(testFilesOK[k])
		if err != nil {
			t.Error(err)
		}
		defer r.Close()

		Df, err := ParseDefinitionFile(r)
		if err != nil {
			t.Log(err)
			t.Fail()
		}
		// Write Deffile output to file
		Df.WriteDefinitionFile(f)
		// And....compare the output (fingers crossed)
		if !compareFiles(t, resultFile[k], f.Name()) {
			t.Logf("<=\tFailed to parse Deffinition file:\t[%s]", k)
			t.Fail()
		}
	}

	// Loop through the Deffiles BAD (must return error)
	for k, v := range testFilesBAD {
		t.Logf("=>\tRunning test for Bad Deffile:\t\t[%s]", k)
		r, err := os.Open(v)
		if err != nil {
			t.Error(err)
		}
		defer r.Close()

		// Parse must return err and a nil Definition struct
		_, err = ParseDefinitionFile(r)
		if err == nil {
			t.Logf("<=\tFailed to parse Bad Deffinition file:\t[%s]", k)
			t.Log(err)
			t.Fail()
		}
	}
}

// compareFiles is a helper func to compare outputs
func compareFiles(t *testing.T, resultFile, testFile string) bool {
	rfile, err := os.Open(resultFile)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	defer rfile.Close()

	tfile, err := os.Open(testFile)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	defer rfile.Close()

	testDef, err := ParseDefinitionFile(tfile)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	rDef, err := ParseDefinitionFile(rfile)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	return reflect.DeepEqual(testDef, rDef)
}

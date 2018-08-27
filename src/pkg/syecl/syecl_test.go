// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package syecl

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

const (
	KeyFP1 = "5994BE54C31CF1B5E1994F987C52CF6D055F072B"
	KeyFP2 = "7064B1D6EFF01B1262FED3F03581D99FE87EAFD1"

	srcContainer1 = "testdata/container1.sif"
	srcContainer2 = "testdata/container2.sif"
	srcContainer3 = "testdata/container3.sif"
)

var testEclConfig = EclConfig{
	Activated: true,
	ExecGroups: []execgroup{
		{"group1", "whitelist", "", []string{KeyFP1, KeyFP2}},
		{"group2", "whitelist", "", []string{KeyFP2}},
	},
}

var testEclFileName string // pathname of the Ecl config file
var testEclDirPath1 string // dirname of the first Ecl execgroup
var testEclDirPath2 string // dirname of the second Ecl execgroup
var testContainer1 string  // pathname of the first test container
var testContainer2 string  // pathname of the second test container
var testContainer3 string  // pathname of the third test container

func TestAPutConfig(t *testing.T) {
	err := PutConfig(testEclConfig, testEclFileName)
	if err != nil {
		t.Error(`PutConfig(config, name):`, err)
	}
}

func TestLoadConfig(t *testing.T) {
	ecl, err := LoadConfig(testEclFileName)
	if err != nil {
		t.Error(`LoadConfig(testEclFileName):`, err)
	}
	if ecl.Activated == false {
		t.Error("the ECL should be activated")
	}
	if ecl.ExecGroups[0].DirPath != testEclDirPath1 {
		t.Error("the path was expected to be:", testEclDirPath1)
	}
	if ecl.ExecGroups[0].KeyFPs[0] != KeyFP1 {
		t.Error("the entity was expected to be:", KeyFP1)
	}
}

func TestValidateConfig(t *testing.T) {
	ecl, err := LoadConfig(testEclFileName)
	if err != nil {
		t.Error(`LoadConfig(testEclFileName):`, err)
	}

	if err = ecl.ValidateConfig(); err != nil {
		t.Error(`ecl.ValidateConfig():`, err)
	}
}

func TestShouldRun(t *testing.T) {
	ecl, err := LoadConfig(testEclFileName)
	if err != nil {
		t.Error(`LoadConfig(testEclFileName):`, err)
	}

	if err = ecl.ValidateConfig(); err != nil {
		t.Error(`ecl.ValidateConfig():`, err)
	}

	// check container1 authorization
	run, err := ecl.ShouldRun(testContainer1)
	if err != nil {
		t.Error(`ecl.ShouldRun(testContainer1):`, err)
	}
	if !run {
		t.Error(testContainer1, "should be allowed to run")
	}
	// check container2 authorization
	run, err = ecl.ShouldRun(testContainer2)
	if err != nil {
		t.Error(`ecl.ShouldRun(testContainer2):`, err)
	}
	if !run {
		t.Error(testContainer2, "should be allowed to run")
	}
	// check container3 authorization (fails with KeyFP)
	run, err = ecl.ShouldRun(testContainer3)
	if err == nil || run == true {
		t.Error(testContainer3, "should NOT be allowed to run")
	}
	// check srcContainer1 authorization (fails with dirpath)
	run, err = ecl.ShouldRun(srcContainer1)
	if err == nil || run == true {
		t.Error(srcContainer1, "should NOT be allowed to run")
	}
}

func copyFile(dst, src string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}

	if err := d.Close(); err != nil {
		return err
	}

	return nil
}

func setup() error {
	// Use TempFile to create a placeholder for the ECL config test file
	tmpfile, err := ioutil.TempFile("", "eclconfig-test")
	if err != nil {
		return nil
	}
	testEclFileName = tmpfile.Name()
	tmpfile.Close()

	// Create two directories where we put test containers
	testEclDirPath1, err = ioutil.TempDir("", "ecldir1")
	if err != nil {
		return err
	}

	testEclDirPath2, err = ioutil.TempDir("", "ecldir2")
	if err != nil {
		return err
	}

	// Set the just created Dirpaths in the EclConfig struct to marshal
	testEclConfig.ExecGroups[0].DirPath = testEclDirPath1
	testEclConfig.ExecGroups[1].DirPath = testEclDirPath2

	// prepare and copy test containers from testdata/* to their test dirpaths
	testContainer1 = filepath.Join(testEclDirPath1, filepath.Base(srcContainer1))
	if err := copyFile(testContainer1, srcContainer1); err != nil {
		return err
	}
	testContainer2 = filepath.Join(testEclDirPath2, filepath.Base(srcContainer2))
	if err := copyFile(testContainer2, srcContainer2); err != nil {
		return err
	}
	testContainer3 = filepath.Join(testEclDirPath2, filepath.Base(srcContainer3))
	if err := copyFile(testContainer3, srcContainer3); err != nil {
		return err
	}

	return nil
}

func shutdown() {
	os.Remove(testEclFileName)
	os.RemoveAll(testEclDirPath1)
	os.RemoveAll(testEclDirPath2)
}

func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		shutdown()
		os.Exit(2)
	}
	ret := m.Run()
	shutdown()
	os.Exit(ret)
}

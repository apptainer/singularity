// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package wlconf

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
)

var testWlConfig = WlConfig{
	Activated: true,
	AuthList: []authorized{{"", []string{KeyFP1, KeyFP2}},
		{"", []string{KeyFP2}},
	},
}

var testWlFileName string // pathname of the whitelist config file
var testWlDomain1 string  // dirname of the first whitelist domaine
var testWlDomain2 string  // dirname of the second whitelist domaine
var testContainer1 string // pathname of the first test container

func TestAPutConfig(t *testing.T) {
	err := PutConfig(testWlConfig, testWlFileName)
	if err != nil {
		t.Error(`PutConfig(config, name):`, err)
	}
}

func TestLoadConfig(t *testing.T) {
	wlcfg, err := LoadConfig(testWlFileName)
	if err != nil {
		t.Error(`LoadConfig(testWlFileName):`, err)
	}
	if wlcfg.Activated == false {
		t.Error("the whitelist should be activated")
	}
	if wlcfg.AuthList[0].Path != testWlDomain1 {
		t.Error("the path was expected to be:", testWlDomain1)
	}
	if wlcfg.AuthList[0].Entities[0] != KeyFP1 {
		t.Error("the entity was expected to be:", KeyFP1)
	}
}

func TestValidateConfig(t *testing.T) {
	wlcfg, err := LoadConfig(testWlFileName)
	if err != nil {
		t.Error(`LoadConfig(testWlFileName):`, err)
	}

	if err = wlcfg.ValidateConfig(); err != nil {
		t.Error(`wlcfg.ValidateConfig():`, err)
	}
}

func TestShouldRun(t *testing.T) {
	wlcfg, err := LoadConfig(testWlFileName)
	if err != nil {
		t.Error(`LoadConfig(testWlFileName):`, err)
	}

	if err = wlcfg.ValidateConfig(); err != nil {
		t.Error(`wlcfg.ValidateConfig():`, err)
	}

	run, err := wlcfg.ShouldRun(testContainer1)
	if err != nil {
		t.Error(`wlcfg.ShouldRun(testContainer1):`, err)
	}

	if !run {
		t.Error(testContainer1, "should be allowed to run")
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
	// Use TempFile to create a placeholder for the wlconfig test file
	tmpfile, err := ioutil.TempFile("", "wlconfig-test")
	if err != nil {
		return nil
	}
	testWlFileName = tmpfile.Name()
	tmpfile.Close()

	// Create two directories (domains) where we put test containers
	testWlDomain1, err = ioutil.TempDir("", "wldir1")
	if err != nil {
		return err
	}

	testWlDomain2, err = ioutil.TempDir("", "wldir2")
	if err != nil {
		return err
	}

	// Set the domain just created paths in the wlConfig struct to marshal
	testWlConfig.AuthList[0].Path = testWlDomain1
	testWlConfig.AuthList[1].Path = testWlDomain2

	// prepare and copy test containers from testdata/* to their test domaines
	testContainer1 = filepath.Join(testWlDomain1, filepath.Base(srcContainer1))
	if err := copyFile(testContainer1, srcContainer1); err != nil {
		return err
	}

	return nil
}

func shutdown() {
	os.Remove(testWlFileName)
	os.RemoveAll(testWlDomain1)
	os.RemoveAll(testWlDomain2)
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

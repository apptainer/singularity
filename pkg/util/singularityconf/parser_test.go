// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularityconf

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestGenerate(t *testing.T) {
	discard := ioutil.Discard

	if err := Generate(discard, "/non-existent/template", nil); err == nil {
		t.Fatalf("unexpected success with non-existent template")
	}
	if err := Generate(discard, "", nil); err == nil {
		t.Fatalf("unexpected success with nil config")
	}
}

func TestParser(t *testing.T) {
	f, err := ioutil.TempFile("", "singularity.conf-")
	if err != nil {
		t.Fatalf("failed to create temporary configuration file: %s", err)
	}
	configFile := f.Name()
	defer os.Remove(configFile)

	defaultConfig, err := GetConfig(nil)
	if err != nil {
		t.Fatalf("failed to get the default configuration: %s", err)
	}

	if err := Generate(f, "", defaultConfig); err != nil {
		t.Fatalf("failed to generate default configuration: %s", err)
	}

	f.Close()

	if _, err = Parse("test_samples/no.conf"); err == nil {
		t.Errorf("unexpected success while opening non existent configuration file")
	}

	config, err := Parse(configFile)
	if err != nil {
		t.Errorf("unexpected error while parsing %s: %s", configFile, err)
	}

	if !reflect.DeepEqual(config, defaultConfig) {
		t.Errorf("config != defaultConfig")
	}

	config, err = Parse("")
	if err != nil {
		t.Errorf("unexpected error while parsing %s: %s", configFile, err)
	}

	if !reflect.DeepEqual(config, defaultConfig) {
		t.Errorf("parsed configuration doesn't match the default configuration")
	}
}

type faultyReader struct {
	io.Reader
}

func (f *faultyReader) Read([]byte) (int, error) {
	return 0, fmt.Errorf("faulty read")
}

func TestGetDirectives(t *testing.T) {
	emptyDirectives := make(Directives)

	faulty := new(faultyReader)
	if _, err := GetDirectives(faulty); err == nil {
		t.Fatalf("unexpected success while getting directives from faulty reader")
	}

	directives, err := GetDirectives(nil)
	if err != nil {
		t.Fatalf("unexpected error while getting directives from nil reader: %s", err)
	}

	if !reflect.DeepEqual(directives, emptyDirectives) {
		t.Errorf("parsed configuration doesn't match the default configuration")
	}
}

func TestGetConfig(t *testing.T) {
	directives := make(Directives)

	directives["allow setuid"] = []string{"bad"}

	if _, err := GetConfig(directives); err == nil {
		t.Errorf("unexpected success while getting config with bad value")
	}

	directives["allow setuid"] = []string{"no"}
	directives["mount dev"] = []string{"bad"}

	if _, err := GetConfig(directives); err == nil {
		t.Errorf("unexpected success while getting config with bad value")
	}

	directives["max loop devices"] = []string{"-42"}
	directives["mount dev"] = []string{"minimal"}

	if _, err := GetConfig(directives); err == nil {
		t.Errorf("unexpected success while getting config with bad value")
	}

	directives["max loop devices"] = []string{"42"}
	directives["bind path"] = []string{"/etc/hosts"}

	directives["download concurrency"] = []string{"42"}
	directives["download part size"] = []string{"1234"}
	directives["download buffer size"] = []string{"4567"}

	config, err := GetConfig(directives)
	if err != nil {
		t.Errorf("unexpected error while getting config: %s", err)
	}
	if config.AllowSetuid != false {
		t.Errorf("bad value for AllowSetuid: %v", config.AllowSetuid)
	}
	if config.MaxLoopDevices != 42 {
		t.Errorf("bad value for MaxLoopDevices: %v", config.MaxLoopDevices)
	}
	if config.MountDev != "minimal" {
		t.Errorf("bad value for MountDev: %v", config.MountDev)
	}
	if !reflect.DeepEqual(config.BindPath, directives["bind path"]) {
		t.Errorf("bad value for BindPath: %v", config.BindPath)
	}
	if config.DownloadConcurrency != 42 {
		t.Errorf("bad value for DownloadConcurrency: %v", config.DownloadConcurrency)
	}
	if config.DownloadPartSize != 1234 {
		t.Errorf("bad value for DownloadPartSize: %v", config.DownloadPartSize)
	}
	if config.DownloadBufferSize != 4567 {
		t.Errorf("bad value for DownloadBufferSize: %v", config.DownloadPartSize)
	}
}

func TestHasDirective(t *testing.T) {
	if HasDirective("") {
		t.Errorf("empty directive should return false")
	}
	if !HasDirective("bind path") {
		t.Errorf("'bind path' should be present")
	}
	if HasDirective("fake directive") {
		t.Errorf("'fake directive' should not be present")
	}
}

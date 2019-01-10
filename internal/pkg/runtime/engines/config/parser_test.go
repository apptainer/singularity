// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

type testConfig struct {
	BoolYes            bool     `default:"yes" authorized:"yes,no" directive:"bool_yes"`
	BoolNo             bool     `default:"no" authorized:"yes,no" directive:"bool_no"`
	Uint               uint     `default:"0" directive:"uint"`
	Int                int      `default:"-0" directive:"int"`
	String             string   `directive:"string"`
	StringAuthorized   string   `authorized:"value1,value2" directive:"string_authorized"`
	StringSlice        []string `directive:"string_slice"`
	StringSliceDefault []string `default:"value1,value2" directive:"string_slice_default"`
}

func genConfig(content []byte) (string, error) {
	f, err := ioutil.TempFile("", "parser-")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(content); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func TestParser(t *testing.T) {
	var def testConfig
	var valid testConfig

	if err := Parser("test_samples/no.conf", &def); err == nil {
		t.Errorf("unexpected success while opening non existent configuration file")
	}

	if err := Parser("", &def); err != nil {
		t.Error(err)
	}
	if def.BoolYes != true {
		t.Errorf("unexpected value for bool_yes: %v", def.BoolYes)
	}
	if def.BoolNo != false {
		t.Errorf("unexpected value for bool_no: %v", def.BoolNo)
	}
	if def.Uint != 0 {
		t.Errorf("unexpected value for uint: %v", def.Uint)
	}
	if def.Int != 0 {
		t.Errorf("unexpected value for int: %v", def.Int)
	}
	if def.String != "" {
		t.Errorf("unexpected value for string: %v", def.String)
	}
	if def.StringAuthorized != "" {
		t.Errorf("unexpected value for string_authorized: %v", def.StringAuthorized)
	}
	if !reflect.DeepEqual(def.StringSlice, []string{}) {
		t.Errorf("unexpected value for string_slice: %v", def.StringSlice)
	}
	if !reflect.DeepEqual(def.StringSliceDefault, []string{"value1", "value2"}) {
		t.Errorf("unexpected value for string_slice_default: %v", def.StringSliceDefault)
	}

	validConfig := []byte(`
		bool_yes = no
		bool_no = yes
		uint = 1
		int = -1
		string = data
		string_authorized = value2
		string_slice = value1
		string_slice = value2
		string_slice = value3
		string_slice_default = value3
	`)

	path, err := genConfig(validConfig)
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(path)

	if err := Parser(path, &valid); err != nil {
		t.Error(err)
	}
	if valid.BoolYes != false {
		t.Errorf("unexpected value for bool_yes: %v", valid.BoolYes)
	}
	if valid.BoolNo != true {
		t.Errorf("unexpected value for bool_no: %v", valid.BoolNo)
	}
	if valid.Uint != 1 {
		t.Errorf("unexpected value for uint: %v", valid.Uint)
	}
	if valid.Int != -1 {
		t.Errorf("unexpected value for int: %v", valid.Int)
	}
	if valid.String != "data" {
		t.Errorf("unexpected value for string: %v", valid.String)
	}
	if valid.StringAuthorized != "value2" {
		t.Errorf("unexpected value for string_authorized: %v", valid.StringAuthorized)
	}
	if !reflect.DeepEqual(valid.StringSlice, []string{"value1", "value2", "value3"}) {
		t.Errorf("unexpected value for string_slice: %v", valid.StringSlice)
	}
	if !reflect.DeepEqual(valid.StringSliceDefault, []string{"value3"}) {
		t.Errorf("unexpected value for string_slice_default: %v", valid.StringSliceDefault)
	}

	for _, s := range []string{
		"bool_yes = enable",
		"bool_no = disable",
		"uint = -1",
		"int = string",
		"string_authorized = value3",
	} {
		badConfig := []byte(s)

		path, err = genConfig(badConfig)
		if err != nil {
			t.Error(err)
		}

		if err := Parser(path, &valid); err == nil {
			t.Errorf("unexpected success while parsing %s", s)
		}

		os.Remove(path)
	}
}

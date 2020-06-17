// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package syecl

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gotest.tools/v3/golden"
)

const (
	KeyFP1 = "5994BE54C31CF1B5E1994F987C52CF6D055F072B"
	KeyFP2 = "7064B1D6EFF01B1262FED3F03581D99FE87EAFD1"
)

var (
	srcContainer1 = filepath.Join("testdata", "container1.sif")
	srcContainer2 = filepath.Join("testdata", "container2.sif")
	srcContainer3 = filepath.Join("testdata", "container3.sif")
)

var (
	testEclDirPath1 string // dirname of the first Ecl execgroup
	testEclDirPath2 string // dirname of the second Ecl execgroup
	testEclDirPath3 string // dirname of the third Ecl execgroup
	testContainer1  string // pathname of the first test container
	testContainer2  string // pathname of the second test container
	testContainer3  string // pathname of the third test container
	testContainer4  string // pathname of the forth test container
)

func TestAPutConfig(t *testing.T) {
	wl := execgroup{
		TagName:  "name",
		ListMode: "whitelist",
		DirPath:  "/var/data1",
		KeyFPs:   []string{KeyFP1, KeyFP2},
	}
	wls := execgroup{
		TagName:  "name",
		ListMode: "whitestrict",
		DirPath:  "/var/data2",
		KeyFPs:   []string{KeyFP1, KeyFP2},
	}
	bl := execgroup{
		TagName:  "name",
		ListMode: "blacklist",
		DirPath:  "/var/data3",
		KeyFPs:   []string{KeyFP1},
	}

	tests := []struct {
		name string
		c    EclConfig
	}{
		{
			name: "Deactivated",
			c:    EclConfig{Activated: false},
		},
		{
			name: "Activated",
			c:    EclConfig{Activated: true},
		},
		{
			name: "WhiteList",
			c:    EclConfig{Activated: true, ExecGroups: []execgroup{wl}},
		},
		{
			name: "WhiteStrict",
			c:    EclConfig{Activated: true, ExecGroups: []execgroup{wls}},
		},
		{
			name: "BlackList",
			c:    EclConfig{Activated: true, ExecGroups: []execgroup{bl}},
		},
		{
			name: "KitchenSink",
			c:    EclConfig{Activated: true, ExecGroups: []execgroup{wl, wls, bl}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tf, err := ioutil.TempFile("", "eclconfig-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tf.Name())
			tf.Close()

			if err := PutConfig(tt.c, tf.Name()); err != nil {
				t.Fatal(err)
			}

			b, err := ioutil.ReadFile(tf.Name())
			if err != nil {
				t.Fatal(err)
			}

			filename := path.Join(strings.Split(t.Name(), "/")...) + ".golden"
			golden.AssertBytes(t, b, filename)
		})
	}
}

func TestLoadConfig(t *testing.T) {
	wl := execgroup{
		TagName:  "name",
		ListMode: "whitelist",
		DirPath:  "/var/data1",
		KeyFPs:   []string{KeyFP1, KeyFP2},
	}
	wls := execgroup{
		TagName:  "name",
		ListMode: "whitestrict",
		DirPath:  "/var/data2",
		KeyFPs:   []string{KeyFP1, KeyFP2},
	}
	bl := execgroup{
		TagName:  "name",
		ListMode: "blacklist",
		DirPath:  "/var/data3",
		KeyFPs:   []string{KeyFP1},
	}

	tests := []struct {
		name       string
		path       string
		wantConfig EclConfig
	}{
		{
			name:       "Deactivated",
			wantConfig: EclConfig{Activated: false},
		},
		{
			name:       "Activated",
			wantConfig: EclConfig{Activated: true},
		},
		{
			name:       "WhiteList",
			wantConfig: EclConfig{Activated: true, ExecGroups: []execgroup{wl}},
		},
		{
			name:       "WhiteStrict",
			wantConfig: EclConfig{Activated: true, ExecGroups: []execgroup{wls}},
		},
		{
			name:       "BlackList",
			wantConfig: EclConfig{Activated: true, ExecGroups: []execgroup{bl}},
		},
		{
			name:       "KitchenSink",
			wantConfig: EclConfig{Activated: true, ExecGroups: []execgroup{wl, wls, bl}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("testdata", "input", fmt.Sprintf("%s.toml", tt.name))
			c, err := LoadConfig(path)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(c, tt.wantConfig) {
				t.Errorf("got config %v, want %v", c, tt.wantConfig)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	wl := execgroup{
		TagName:  "name",
		ListMode: "whitelist",
		DirPath:  testEclDirPath1,
		KeyFPs:   []string{KeyFP1, KeyFP2},
	}
	wls := execgroup{
		TagName:  "name",
		ListMode: "whitestrict",
		DirPath:  testEclDirPath2,
		KeyFPs:   []string{KeyFP1, KeyFP2},
	}
	bl := execgroup{
		TagName:  "name",
		ListMode: "blacklist",
		DirPath:  testEclDirPath3,
		KeyFPs:   []string{KeyFP1},
	}

	tests := []struct {
		name    string
		c       EclConfig
		wantErr bool
	}{
		{
			name: "DuplicatePaths",
			c: EclConfig{ExecGroups: []execgroup{
				{DirPath: "/var/data"},
				{DirPath: "/var/data"},
			}},
			wantErr: true,
		},
		{
			name: "RelativePath",
			c: EclConfig{ExecGroups: []execgroup{
				{DirPath: "testdata"},
			}},
			wantErr: true,
		},
		{
			name: "BadMode",
			c: EclConfig{ExecGroups: []execgroup{
				{ListMode: "bad"},
			}},
			wantErr: true,
		},
		{
			name: "BadMode",
			c: EclConfig{ExecGroups: []execgroup{
				{ListMode: "whitelist", KeyFPs: []string{"bad"}},
			}},
			wantErr: true,
		},
		{
			name: "Deactivated",
			c:    EclConfig{Activated: false},
		},
		{
			name: "Activated",
			c:    EclConfig{Activated: true},
		},
		{
			name: "WhiteList",
			c:    EclConfig{Activated: true, ExecGroups: []execgroup{wl}},
		},
		{
			name: "WhiteStrict",
			c:    EclConfig{Activated: true, ExecGroups: []execgroup{wls}},
		},
		{
			name: "BlackList",
			c:    EclConfig{Activated: true, ExecGroups: []execgroup{bl}},
		},
		{
			name: "KitchenSink",
			c:    EclConfig{Activated: true, ExecGroups: []execgroup{wl, wls, bl}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.c.ValidateConfig(); (err != nil) != tt.wantErr {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldRun(t *testing.T) {
	eclDeactivated := EclConfig{Activated: false}
	eclBadListMode := EclConfig{
		Activated: true,
		ExecGroups: []execgroup{
			{
				ListMode: "bad",
			},
		},
	}
	eclDirPath := EclConfig{
		Activated: true,
		ExecGroups: []execgroup{
			{
				TagName:  "group1",
				ListMode: "whitelist",
				DirPath:  testEclDirPath1,
				KeyFPs:   []string{KeyFP1, KeyFP2},
			},
			{
				TagName:  "group2",
				ListMode: "whitestrict",
				DirPath:  testEclDirPath2,
				KeyFPs:   []string{KeyFP1, KeyFP2},
			},
			{
				TagName:  "group3",
				ListMode: "blacklist",
				DirPath:  testEclDirPath3,
				KeyFPs:   []string{KeyFP1},
			},
		},
	}
	eclNoDirPath := EclConfig{
		Activated: true,
		ExecGroups: []execgroup{
			{
				TagName:  "group1",
				ListMode: "whitelist",
				DirPath:  "",
				KeyFPs:   []string{KeyFP1, KeyFP2},
			},
		},
	}

	tests := []struct {
		name    string
		c       EclConfig
		path    string
		wantRun bool
		wantErr bool
	}{
		{"BadListMode", eclBadListMode, testContainer1, false, true},
		{"Deactivated", eclDeactivated, testContainer1, true, false},
		{"DirPathTestContainer1", eclDirPath, testContainer1, true, false},
		{"DirPathTestContainer2", eclDirPath, testContainer2, true, false},
		{"DirPathTestContainer3", eclDirPath, testContainer3, false, true},
		{"DirPathTestContainer4", eclDirPath, testContainer4, false, true},
		{"DirPathSrcContainer1", eclDirPath, srcContainer1, false, true},
		{"DirPathSrcContainer2", eclDirPath, srcContainer2, false, true},
		{"DirPathSrcContainer3", eclDirPath, srcContainer3, false, true},
		{"NoDirPathSrcContainer1", eclNoDirPath, srcContainer1, true, false},
		{"NoDirPathSrcContainer2", eclNoDirPath, srcContainer2, true, false},
		{"NoDirPathSrcContainer3", eclNoDirPath, srcContainer3, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test ShouldRun (takes path).
			got, err := tt.c.ShouldRun(tt.path)

			if got != tt.wantRun {
				t.Errorf("got run %v, want %v", got, tt.wantRun)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("got err %v, wantErr %v", err, tt.wantErr)
			}

			// Test ShouldRun (takes file descriptor).
			f, err := os.Open(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			got, err = tt.c.ShouldRunFp(f)

			if got != tt.wantRun {
				t.Errorf("got run %v, want %v", got, tt.wantRun)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("got err %v, wantErr %v", err, tt.wantErr)
			}
		})
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

	return d.Close()
}

func setup() error {
	var err error

	// Create three directories where we put test containers
	testEclDirPath1, err = ioutil.TempDir("", "ecldir1-")
	if err != nil {
		return err
	}

	testEclDirPath2, err = ioutil.TempDir("", "ecldir2-")
	if err != nil {
		return err
	}

	testEclDirPath3, err = ioutil.TempDir("", "ecldir3-")
	if err != nil {
		return err
	}

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
	testContainer4 = filepath.Join(testEclDirPath3, filepath.Base(srcContainer3))
	return copyFile(testContainer4, srcContainer3)
}

func shutdown() {
	os.RemoveAll(testEclDirPath1)
	os.RemoveAll(testEclDirPath2)
	os.RemoveAll(testEclDirPath3)
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

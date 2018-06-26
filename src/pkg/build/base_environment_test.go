// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"io/ioutil"
	"os"
	"testing"
)

func testWithGoodDir(t *testing.T, f func(d, d2 string) error) {
	d, err := ioutil.TempDir(os.TempDir(), "test")
	d2, err := ioutil.TempDir(os.TempDir(), "test")
	if err != nil {
		t.Fatalf("Failed to make temporary directory: %v", err)
	}
	defer os.RemoveAll(d)
	defer os.RemoveAll(d2)

	if err := f(d, d2); err != nil {
		t.Fatalf("Unexpected failure: %v", err)
	}
}

func testWithBadDir(t *testing.T, f func(d, d2 string) error) {
	if err := f("/this/will/be/a/problem", "/this/will/be/a/problem2"); err == nil {
		t.Fatalf("Unexpected success with bad directory")
	}
}

func TestMakeDirs(t *testing.T) {
	testWithGoodDir(t, makeDirs)
	testWithBadDir(t, makeDirs)
}

func TestMakeFiles(t *testing.T) {
	testWithGoodDir(t, func(d, d2 string) error {
		if err := makeDirs(d, d2); err != nil {
			return err
		}
		return makeFiles(d, d2)
	})
	testWithBadDir(t, makeFiles)
}

func TestMakeBaseEnv(t *testing.T) {
	testWithGoodDir(t, makeBaseEnv)
	testWithBadDir(t, makeBaseEnv)
}

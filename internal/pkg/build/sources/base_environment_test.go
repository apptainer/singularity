// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func testWithGoodDir(t *testing.T, f func(d string) error) {
	d, err := ioutil.TempDir(os.TempDir(), "test")
	if err != nil {
		t.Fatalf("Failed to make temporary directory: %v", err)
	}
	defer os.RemoveAll(d)

	if err := f(d); err != nil {
		t.Fatalf("Unexpected failure: %v", err)
	}
}

func testWithBadDir(t *testing.T, f func(d string) error) {
	if err := f("/this/will/be/a/problem"); err == nil {
		t.Fatalf("Unexpected success with bad directory")
	}
}

func TestMakeDirs(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	testWithGoodDir(t, makeDirs)
	testWithBadDir(t, makeDirs)
}

func TestMakeSymlinks(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	testWithGoodDir(t, makeSymlinks)
	testWithBadDir(t, makeSymlinks)
}

func TestMakeFiles(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	testWithGoodDir(t, func(d string) error {
		if err := makeDirs(d); err != nil {
			return err
		}
		return makeFiles(d)
	})
	testWithBadDir(t, makeFiles)
}

func TestMakeBaseEnv(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	testWithGoodDir(t, makeBaseEnv)
	testWithBadDir(t, makeBaseEnv)
}

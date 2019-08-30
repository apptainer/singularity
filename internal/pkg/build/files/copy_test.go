// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package files

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var sourceFileContent = "Source File Content\n"

func TestMakeParentDir(t *testing.T) {
	tests := []struct {
		name   string
		srcNum int
		path   string
		parent bool // this specifies if the correct path should have the full path created or just the parent
	}{
		{
			name:   "basic",
			srcNum: 1,
			path:   "basic/path",
			parent: true,
		},
		{
			name:   "trailing slash",
			srcNum: 1,
			path:   "trailing/slash/",
			parent: false,
		},
		{
			name:   "multiple",
			srcNum: 2,
			path:   "multiple/files",
			parent: false,
		},
		{
			name:   "multiple trailing slash",
			srcNum: 2,
			path:   "multiple/trailing/slash/",
			parent: false,
		},
		{
			name:   "exists",
			srcNum: 1,
			path:   "", // this will create a path of just the testdir, which will always exist
			parent: false,
		},
		{
			name:   "exists multiple",
			srcNum: 2,
			path:   "", // this will create a path of just the testdir, which will always exist
			parent: false,
		},
	}

	// while running tetss, make sure to remove everything past the tmp dir created so tests to accidentially collide
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create tmpdir for each test
			dir, err := ioutil.TempDir("", "parent-dir-test-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(dir)

			// concatenate test path with directory, do not use a join function so that we do not remove a trailing slash
			path := dir + "/" + tt.path
			if err := makeParentDir(path, tt.srcNum); err != nil {
				t.Errorf("")
			}

			clean := filepath.Clean(path)
			if tt.parent {
				// full path should not exist
				_, err := os.Stat(clean)
				if !os.IsNotExist(err) {
					t.Errorf("full path created when only parent should have been made")
				}

				// parent should exist
				_, err = os.Stat(filepath.Dir(clean))
				if os.IsNotExist(err) {
					t.Errorf("parent not created when it should have been made")
				}
			} else {
				// full path should exist
				_, err := os.Stat(clean)
				if os.IsNotExist(err) {
					t.Errorf("full path not created when it should have been made")
				}
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	// create tmpdir
	dir, err := ioutil.TempDir("", "copy-test-src-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// prep src file to copy
	srcFile := filepath.Join(dir, "sourceFile")
	if err := ioutil.WriteFile(srcFile, []byte(sourceFileContent), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		src       string
		dst       string
		finalpath string
	}{
		{"ToDir", srcFile, "", "sourceFile"},
		{"ToDirSlash", srcFile, "destDir/", "destDir/sourceFile"},
		{"ToFile", srcFile, "destDir/destFile", "destDir/destFile"},
		{"LongPathToFile", srcFile, "destDir/long/path/to/destFile", "destDir/long/path/to/destFile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create tmpdir
			dstDir, err := ioutil.TempDir("", "copy-test-dst-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(dstDir)

			// manually concatenating because I don't want a Join function to clean the trailing slash
			dst := dstDir + "/" + tt.dst
			if err := Copy(tt.src, dst); err != nil {
				t.Errorf("unexpected failure running %s test: %s", t.Name(), err)
			}

			dstFinal := filepath.Join(dstDir, tt.finalpath)
			// verify file was copied
			_, err = os.Stat(dstFinal)
			if os.IsNotExist(err) {
				t.Errorf("failure to correctly copy file %s test: %s", t.Name(), err)
			}

			// verify file contents
			content, err := ioutil.ReadFile(dstFinal)
			if err != nil {
				t.Errorf("unexpected failure reading file %s test: %s", t.Name(), err)
			}
			if string(content) != sourceFileContent {
				t.Errorf("failure reading file %s test: %s", t.Name(), err)
			}
		})
	}
}

func TestCopyDir(t *testing.T) {
	// create tmpdir
	dir, err := ioutil.TempDir("", "copy-test-src-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// prep src dir to copy
	srcDir := filepath.Join(dir, "sourceDir")
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	// prep src file
	srcFile := filepath.Join(srcDir, "sourceFile")
	if err := ioutil.WriteFile(srcFile, []byte(sourceFileContent), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		src       string
		dst       string
		finalpath string
	}{
		{"ToDir", srcDir, "destDir", "destDir"},
		{"ToDirSlash", srcDir, "destDir/", "destDir/sourceDir"},
		{"LongPathToDir", srcDir, "long/path/to/destDir", "long/path/to/destDir"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create tmpdir
			dstDir, err := ioutil.TempDir("", "copy-test-dst-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(dstDir)

			// manually concatenating because I don't want a Join function to clean the trailing slash
			dst := dstDir + "/" + tt.dst
			if err := Copy(tt.src, dst); err != nil {
				t.Errorf("unexpected failure running %s test: %s", t.Name(), err)
			}

			dstFinal := filepath.Join(dstDir, tt.finalpath)
			// verify file was copied
			f, err := os.Stat(dstFinal)
			if os.IsNotExist(err) {
				t.Errorf("failure to correctly copy dir %s test: %s", t.Name(), err)
			} else if !f.IsDir() {
				t.Errorf("failure to correctly copy dir %s test: dst is not a dir", t.Name())
			}

			// verify file contents
			content, err := ioutil.ReadFile(filepath.Join(dstFinal, "sourceFile"))
			if err != nil {
				t.Errorf("unexpected failure reading file %s test: %s", t.Name(), err)
			}
			if string(content) != sourceFileContent {
				t.Errorf("failure reading file %s test: %s", t.Name(), err)
			}
		})
	}
}

func TestCopyFail(t *testing.T) {
	// create tmpdir
	dir, err := ioutil.TempDir("", "copy-test-src")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	tests := []struct {
		name string
		src  string
		dst  string
	}{
		{"NoSrc", filepath.Join(dir, "not/a/file"), "file"},
	}

	for _, tt := range tests {
		// make src and dst relative to tmpdir
		filepath.Join(dir, tt.src)

		t.Run(tt.name, func(t *testing.T) {
			// create tmpdir
			dstDir, err := ioutil.TempDir("", "copy-test-dst-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(dstDir)

			dst := filepath.Join(dstDir, tt.dst)
			if err := Copy(tt.src, dst); err == nil {
				t.Errorf("unexpected success running %s test: %s", t.Name(), err)
			}
		})
	}
}

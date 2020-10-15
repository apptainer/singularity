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

	"github.com/sylabs/singularity/internal/pkg/util/fs"
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

	// while running tests, make sure to remove everything past the tmp dir created so tests to accidentially collide
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
	srcSpaceFile := filepath.Join(dir, "source File")
	if err := ioutil.WriteFile(srcSpaceFile, []byte(sourceFileContent), 0644); err != nil {
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
		{"FromSpace", srcSpaceFile, "", "source File"},
		{"ToSpace", srcFile, "dest File", "dest File"},
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
			if err := Copy(tt.src, dst, false); err != nil {
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
			if err := Copy(tt.src, dst, false); err != nil {
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

func TestCopySymlink(t *testing.T) {
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

	// prep src symlink
	srcLink := filepath.Join(srcDir, "sourceLink")
	if err := os.Symlink(srcFile, srcLink); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		src          string
		dst          string
		finalpath    string
		shouldFollow bool
	}{
		// When copied via traversal the symlink should not be followed
		{"DirectoryNoFollow", srcDir, "destDir/", "destDir/sourceDir/sourceLink", false},
		// When copied as a specified source, the link should be followed
		{"LinkFollow", srcLink, "destDir/", "destDir/sourceLink", true},
		// When copied via a glob pattern that resolves to the link directly the link should be followed
		{"GlobFollow", srcLink, "destDir/", "destDir/sourceLink", true},
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
			if err := Copy(tt.src, dst, false); err != nil {
				t.Errorf("unexpected failure running %s test: %s", t.Name(), err)
			}

			dstFinal := filepath.Join(dstDir, tt.finalpath)
			// verify file was copied
			_, err = os.Stat(dstFinal)
			if os.IsNotExist(err) {
				t.Errorf("failure to copy link %s test: %s", t.Name(), err)
			}

			// check if we have a correctly followed/non-followed link
			if !tt.shouldFollow && !fs.IsLink(dstFinal) {
				t.Errorf("%s should be a symlink", dstFinal)
			}

			if tt.shouldFollow && fs.IsLink(dstFinal) {
				t.Errorf("%s should not be a symlink", dstFinal)
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
			if err := Copy(tt.src, dst, false); err == nil {
				t.Errorf("unexpected success running %s test: %s", t.Name(), err)
			}
		})
	}
}

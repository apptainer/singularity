// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package files

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

/*
files-test-dir/
├── dirL1
│   ├── dirL2
│   │   ├── dirL3
│   │   │   ├── file
│   │   │   └── .file
│   │   ├── .dirL3
│   │   │   ├── file
│   │   │   └── .file
│   │   ├── file
│   │   └── .file
│   ├── .dirL2
│   │   ├── dirL3
│   │   │   ├── file
│   │   │   └── .file
│   │   ├── .dirL3
│   │   │   ├── file
│   │   │   └── .file
│   │   ├── file
│   │   └── .file
│   ├── file
│   └── .file
├── dir\nnewline
│   ├── file
│   ├── .file
│   ├── file\nnewline
│   ├── file space
│   └── file\ttab
├── dir space
│   ├── file
│   ├── .file
│   ├── file\nnewline
│   ├── file space
│   └── file\ttab
├── dir\ttab
│   ├── file
│   ├── .file
│   ├── file\nnewline
│   ├── file space
│   └── file\ttab
├── file
└── .file
*/

func formatSlice(slice []string) (s string) {
	for _, elm := range slice {
		s += strconv.Quote(elm) + "\n"
	}
	return s
}

func contains(slice []string, s string) bool {
	for _, elm := range slice {
		if elm == s {
			return true
		}
	}
	return false
}

func createTestDirLayout(t *testing.T) string {
	testDirName, err := ioutil.TempDir("", "files-test-dir-")
	if err != nil {
		t.Fatalf("cannot create temporary dir for testing: %s\n", err)
	}

	dirList := []string{
		"dirL1",
		"dirL1/dirL2",
		"dirL1/dirL2/dirL3",
		"dirL1/dirL2/.dirL3",
		"dirL1/.dirL2",
		"dirL1/.dirL2/dirL3",
		"dirL1/.dirL2/.dirL3",
		"dir space",
		"dir\ttab",
		"dir\nnewline",
	}

	for _, d := range dirList {
		if err := os.Mkdir(filepath.Join(testDirName, d), 0755); err != nil {
			t.Fatalf("while making directory %s: %s", d, err)
		}
	}

	fileList := []string{
		"file",
		".file",
	}

	err = filepath.Walk(testDirName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}
		if info.IsDir() {
			for _, f := range fileList {
				fd, err := os.Create(filepath.Join(path, f))
				if err != nil {
					t.Fatalf("while making file %s: %s", f, err)
				}
				fd.Close()
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("while walking testdir tree: %s", err)
	}

	specialFileList := []string{
		"file space",
		"file\ttab",
		"file\nnewline",
	}

	specialFileDirs := []string{
		"dir space",
		"dir\ttab",
		"dir\nnewline",
	}
	for _, d := range specialFileDirs {
		for _, f := range specialFileList {
			fd, err := os.Create(filepath.Join(testDirName, d, f))
			if err != nil {
				t.Fatalf("while making special file %s: %s", f, err)
			}
			fd.Close()
		}
	}

	return testDirName
}

func TestExpandPath(t *testing.T) {
	testDir := createTestDirLayout(t)
	defer os.RemoveAll(testDir)

	tests := []struct {
		name    string
		path    string
		correct []string
	}{
		{
			name:    "basic",
			path:    "dirL1",
			correct: []string{"dirL1"},
		},
		{
			name:    "wildcard",
			path:    "dirL1/*",
			correct: []string{"dirL1/dirL2", "dirL1/file"},
		},
		{
			name:    "wildcardFile",
			path:    "dirL1/*/file",
			correct: []string{"dirL1/dirL2/file"},
		},
		{
			name:    "multipleWildcards",
			path:    "dirL1/*/dirL3/*",
			correct: []string{"dirL1/dirL2/dirL3/file"},
		},
		{
			name:    "hiddenFileWildcards",
			path:    "dirL1/.*",
			correct: []string{"dirL1/.", "dirL1/..", "dirL1/.dirL2", "dirL1/.file"},
		},
		{
			name:    "hiddenDirWildcards",
			path:    "dirL1/.*/dirL3/*",
			correct: []string{"dirL1/.dirL2/dirL3/file"},
		},
		{
			name:    "?Wildcards",
			path:    "dirL1/?irL2/dirL3/????",
			correct: []string{"dirL1/dirL2/dirL3/file"},
		},
		{
			name:    "?WildcardNoMatch",
			path:    "?irL1/?",
			correct: []string{"?irL1/?"},
		},
		{
			name:    "PathWhitespace",
			path:    "*",
			correct: []string{"dir\ttab", "dir\nnewline", "dir space", "dirL1", "file"},
		},
		{
			name: "PathAndFilenameWhitespace",
			path: "*/*",
			correct: []string{
				"dir\ttab/file",
				"dir\ttab/file\ttab",
				"dir\ttab/file\nnewline",
				"dir\ttab/file space",
				"dir\nnewline/file",
				"dir\nnewline/file\ttab",
				"dir\nnewline/file\nnewline",
				"dir\nnewline/file space",
				"dir space/file",
				"dir space/file\ttab",
				"dir space/file\nnewline",
				"dir space/file space",
				"dirL1/dirL2",
				"dirL1/file",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// make tt.path relative to testDir
			path := filepath.Join(testDir, tt.path) // + "/" + tt.path
			// run it through wildcard function
			files, err := expandPath(path)
			if err != nil {
				t.Errorf("while expanding path: %s", err)
			}

			// make correct output relative to tmp
			// manually concatenate in order to prevent path cleaning from Join()
			var correct []string
			for _, c := range tt.correct {
				correct = append(correct, testDir+"/"+c)
			}

			for _, c := range correct {
				if !contains(files, c) {
					t.Logf("Generated %d results: %s", len(files), formatSlice(files))
					t.Logf("Correct %d results: %s", len(correct), formatSlice(correct))
					t.Errorf("matched files are not correct")
					break
				}
			}
		})
	}
}

func TestAddPrefix(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		path    string
		correct string
	}{
		{
			name:    "sanity",
			prefix:  "",
			path:    "/some/path",
			correct: "/some/path",
		},
		{
			name:    "basicPrepend",
			prefix:  "/some",
			path:    "/path",
			correct: "/some/path",
		},
		{
			name:    "basicJoinTrailingSlash",
			prefix:  "/some",
			path:    "/path/",
			correct: "/some/path/",
		},
		{
			name:    "manySlashes",
			prefix:  "/some/",
			path:    "//path/to/dest//",
			correct: "/some/path/to/dest/",
		},
		{
			name:    "root",
			prefix:  "",
			path:    "/",
			correct: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// run it through wildcard function
			path := AddPrefix(tt.prefix, tt.path)
			if path != tt.correct {
				t.Errorf("join created incorrect path: %s correct: %s", path, tt.correct)
			}
		})
	}
}

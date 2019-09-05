// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package types

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewBundle(t *testing.T) {
	testDir, err := ioutil.TempDir("", "bundleTest-")
	if err != nil {
		t.Fatal("Could not create temporary directory", err)
	}
	defer os.RemoveAll(testDir)

	tt := []struct {
		name        string
		rootfs      string
		tempDir     string
		expectError string
	}{
		{
			name:        "invalid temp dir",
			rootfs:      filepath.Join(testDir, t.Name()+"-bundle1"),
			tempDir:     "/foo/bar",
			expectError: `could not create temp dir in "/foo/bar": stat /foo/bar: no such file or directory`,
		},
		{
			name:        "all ok",
			rootfs:      filepath.Join(testDir, t.Name()+"-bundle2"),
			tempDir:     testDir,
			expectError: ``,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			b, err := NewBundle(tc.rootfs, tc.tempDir)
			if tc.expectError == "" && err != nil {
				t.Errorf("Expected no error, but got %v", err)
			}
			if tc.expectError != "" {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				} else {
					if !strings.Contains(err.Error(), tc.expectError) {
						t.Errorf("Expected %q, but got %v", tc.expectError, err)
					}
				}
			}

			if b != nil {
				// check if the directories were actually created
				_, err := os.Stat(b.RootfsPath)
				if err != nil {
					t.Errorf("RootfsPath stat failed: %v", err)
				}
				_, err = os.Stat(b.TmpDir)
				if err != nil {
					t.Errorf("TmpDir stat failed: %v", err)
				}

				if err := b.Remove(); err != nil {
					t.Errorf("Could not remove bundle: %v", err)
				}

				// check if the directories were actually removed
				_, err = os.Stat(b.RootfsPath)
				if !os.IsNotExist(err) {
					t.Errorf("RootfsPath was not removed: %v", err)
				}
				_, err = os.Stat(b.TmpDir)
				if !os.IsNotExist(err) {
					t.Errorf("TmpDir was not removed: %v", err)
				}
			}
		})
	}

}

func TestBundle_RunSections(t *testing.T) {
	tt := []struct {
		name      string
		sections  []string
		run       string
		expectRun bool
	}{
		{
			name:      "none",
			sections:  []string{"none"},
			run:       "test",
			expectRun: false,
		},
		{
			name:      "all",
			sections:  []string{"all"},
			run:       "test",
			expectRun: true,
		},
		{
			name:      "not found",
			sections:  []string{"foo", "bar"},
			run:       "test",
			expectRun: false,
		},
		{
			name:      "found",
			sections:  []string{"foo", "test", "bar"},
			run:       "test",
			expectRun: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			b := Bundle{
				Opts: Options{
					Sections: tc.sections,
				},
			}

			actual := b.RunSection(tc.run)
			if actual != tc.expectRun {
				t.Fatalf("Extected %v, but got %v", tc.expectRun, actual)
			}
		})
	}
}

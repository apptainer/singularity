// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package crypt

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/fs/squashfs"
)

func TestEncrypt(t *testing.T) {
	test.EnsurePrivilege(t)
	defer test.ResetPrivilege(t)

	dev := &Device{}

	emptyFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("failed to create temporary file: %s", err)
	}
	err = emptyFile.Close()
	if err != nil {
		t.Fatalf("failed to close file %s: %s", emptyFile.Name(), err)
	}
	defer os.Remove(emptyFile.Name())

	// Create a dummy squashfs file
	dummyDir, err := ioutil.TempDir("", "dummy-fs-")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}
	defer os.RemoveAll(dummyDir) // This is delete the directory and all its sub-directories

	// We create a few more sub-directories; note that they will be
	// removed when the top-directory (dummyDir) will be removed.
	dummyRootDir := filepath.Join(dummyDir, "root")
	err = os.MkdirAll(dummyRootDir, 0755)
	if err != nil {
		t.Fatalf("failed to create %s: %s", dummyRootDir, err)
	}
	dummyRootFile := filepath.Join(dummyRootDir, "EMPTYFILE")
	err = fs.Touch(dummyRootFile)
	if err != nil {
		t.Fatalf("failed to create dummy file %s: %s", dummyRootFile, err)
	}
	squashfsBin, err := squashfs.GetPath()
	if err != nil {
		t.Fatalf("failed to get path to squashfs binary: %s", err)
	}
	tempTargetFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("failed to create temporary file: %s", err)
	}
	err = tempTargetFile.Close()
	if err != nil {
		t.Fatalf("failed to close file %s: %s", tempTargetFile.Name(), err)
	}
	defer os.Remove(tempTargetFile.Name())
	squashfsArgs := []string{dummyDir, tempTargetFile.Name(), "-noappend"}
	cmd := exec.Command(squashfsBin, squashfsArgs...)
	err = cmd.Run()
	if err != nil {
		t.Fatalf("failed to create squashfs file: %s", err)
	}

	tests := []struct {
		name        string
		path        string
		key         []byte
		skipCleanup bool
		shallPass   bool
	}{
		{
			name:      "empty path",
			path:      "",
			key:       []byte("dummyKey"),
			shallPass: false,
		},
		/* FIXME: deactivate because it creates too much variability in test results with CI
		{
			name:      "empty file",
			path:      emptyFile.Name(),
			key:       []byte("dummyKey"),
			shallPass: false,
		},
		*/
		{
			name:      "valid file",
			path:      tempTargetFile.Name(),
			key:       []byte("dummyKey"),
			shallPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devPath, err := dev.EncryptFilesystem(tt.path, tt.key)
			if tt.shallPass && err != nil {
				if err == ErrUnsupportedCryptsetupVersion {
					t.Skip("the version of cryptsetup available is not compatible")
				} else {
					t.Fatalf("test %s expected to succeed but failed: %s", tt.name, err)
				}
			}
			defer os.Remove(devPath)

			if !tt.shallPass && err == nil {
				t.Fatalf("test %s expected to fail but succeeded", tt.name)
			}

			// Clean up successful tests
			if tt.shallPass {
				devName, err := dev.Open(tt.key, devPath)
				if err != nil {
					t.Fatalf("failed to open encrypted device: %s", err)
				}
				err = dev.CloseCryptDevice(devName)
				if err != nil {
					t.Fatalf("failed to close crypt device: %s", err)
				}
			}
		})
	}
}

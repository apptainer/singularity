// Copyright (c) 2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package archive

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/hpcng/singularity/internal/pkg/test"
	"github.com/hpcng/singularity/internal/pkg/util/fs"
)

func TestCopyWithTar(t *testing.T) {
	t.Run("privileged", func(t *testing.T) {
		test.EnsurePrivilege(t)
		testCopyWithTar(t)
	})

	t.Run("unprivileged", func(t *testing.T) {
		test.DropPrivilege(t)
		defer test.ResetPrivilege(t)
		testCopyWithTar(t)
	})
}

func testCopyWithTar(t *testing.T) {
	srcRoot, err := ioutil.TempDir("", "copywithtar-src-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(srcRoot)

	// Source Files
	srcFile := filepath.Join(srcRoot, "srcFile")
	if err := ioutil.WriteFile(srcFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Source Dirs
	srcDir := filepath.Join(srcRoot, "srcDir")
	if err := os.Mkdir(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Source Symlink
	srcLink := filepath.Join(srcRoot, "srcLink")
	if err := os.Symlink("srcFile", srcLink); err != nil {
		t.Fatal(err)
	}

	dstRoot, err := ioutil.TempDir("", "copywithtar-dst-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dstRoot)

	// Perform the actual copy to a subdir of our dst tempdir.
	// This ensures CopyWithTar has to create the dest directory, which is
	// where the non-wrapped call would fail for unprivileged users.
	err = CopyWithTar(srcRoot, path.Join(dstRoot, "dst"))
	if err != nil {
		t.Fatalf("Error during CopyWithTar: %v", err)
	}

	tests := []struct {
		name       string
		expectPath string
		expectFile bool
		expectDir  bool
		expectLink bool
	}{
		{
			name:       "file",
			expectPath: "dst/srcFile",
			expectFile: true,
		},
		{
			name:       "dir",
			expectPath: "dst/srcDir",
			expectDir:  true,
		},
		{
			name:       "symlink",
			expectPath: "dst/srcLink",
			expectFile: true,
			expectLink: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dstFinal := filepath.Join(dstRoot, tt.expectPath)
			// verify file was copied
			_, err = os.Stat(dstFinal)
			if err != nil && !os.IsNotExist(err) {
				t.Fatalf("while checking for destination file: %s", err)
			}
			if os.IsNotExist(err) {
				t.Errorf("expected destination %s does not exist", dstFinal)
			}

			// File when expected?
			if tt.expectFile && !fs.IsFile(dstFinal) {
				t.Errorf("destination %s should be a file, but isn't", dstFinal)
			}
			// Dir when expected?
			if tt.expectDir && !fs.IsDir(dstFinal) {
				t.Errorf("destination %s should be a directory, but isn't", dstFinal)
			}
			// Symlink when expected
			if tt.expectLink && !fs.IsLink(dstFinal) {
				t.Errorf("destination %s should be a symlink, but isn't", dstFinal)
			}
			if !tt.expectLink && fs.IsLink(dstFinal) {
				t.Errorf("destination %s should be a symlink, but is", dstFinal)
			}
		})
	}
}

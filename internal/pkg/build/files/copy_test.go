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

	"github.com/hpcng/singularity/internal/pkg/util/fs"
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
			path:   "basic/path",
			parent: true,
		},
		{
			name:   "trailing slash",
			path:   "trailing/slash/",
			parent: false,
		},
		{
			name:   "exists",
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
			if err := makeParentDir(path); err != nil {
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

// TestCopyFromHost tests that copying non-nested source dirs, files, links to various
// destinations works. CopyFromHost should always resolve symlinks.
func TestCopyFromHost(t *testing.T) {
	// create tmpdir
	dir, err := ioutil.TempDir("", "copy-test-src-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Source Files
	srcFile := filepath.Join(dir, "srcFile")
	if err := ioutil.WriteFile(srcFile, []byte(sourceFileContent), 0644); err != nil {
		t.Fatal(err)
	}
	srcFileGlob := filepath.Join(dir, "srcFi?*")
	srcSpaceFile := filepath.Join(dir, "src File")
	if err := ioutil.WriteFile(srcSpaceFile, []byte(sourceFileContent), 0644); err != nil {
		t.Fatal(err)
	}
	// Source Dirs
	srcDir := filepath.Join(dir, "srcDir")
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcDirGlob := filepath.Join(dir, "srcD?*")
	srcSpaceDir := filepath.Join(dir, "src Dir")
	if err := os.Mkdir(srcSpaceDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcGlob := filepath.Join(dir, "src*")
	// Nested File (to test multi level glob)
	srcFileNested := filepath.Join(dir, "srcDir/srcFileNested")
	if err := ioutil.WriteFile(srcFileNested, []byte(sourceFileContent), 0644); err != nil {
		t.Fatal(err)
	}
	srcFileNestedGlob := filepath.Join(dir, "srcDi?/srcFil?Nested")
	// Source Symlinks
	srcFileLinkAbs := filepath.Join(dir, "srcFileLinkAbs")
	if err := os.Symlink(srcFile, srcFileLinkAbs); err != nil {
		t.Fatal(err)
	}
	srcFileLinkRel := filepath.Join(dir, "srcFileLinkRel")
	if err := os.Symlink("./srcFile", srcFileLinkRel); err != nil {
		t.Fatal(err)
	}
	srcDirLinkAbs := filepath.Join(dir, "srcDirLinkAbs")
	if err := os.Symlink(srcDir, srcDirLinkAbs); err != nil {
		t.Fatal(err)
	}
	srcDirLinkRel := filepath.Join(dir, "srcDirLinkRel")
	if err := os.Symlink("./srcDir", srcDirLinkRel); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		src        string
		dst        string
		expectPath string
		expectFile bool
		expectDir  bool
	}{
		// Source is a file
		{
			name:       "SrcFileNoDest",
			src:        srcFile,
			dst:        "",
			expectPath: srcFile,
			expectFile: true,
		},
		{
			name:       "SrcFileToDir",
			src:        srcFile,
			dst:        "dstDir/",
			expectPath: "dstDir/srcFile",
			expectFile: true,
		},
		{
			name:       "srcFileToFile",
			src:        srcFile,
			dst:        "dstDir/dstFile",
			expectPath: "dstDir/dstFile",
			expectFile: true,
		},
		{
			name:       "srcFileToFileLongPath",
			src:        srcFile,
			dst:        "dstDir/long/path/to/dstFile",
			expectPath: "dstDir/long/path/to/dstFile",
			expectFile: true,
		},
		{
			name:       "srcFileSpace",
			src:        srcSpaceFile,
			dst:        "src File",
			expectPath: "src File",
			expectFile: true,
		},
		{
			name:       "dstFileSpace",
			src:        srcFile,
			dst:        "dst File",
			expectPath: "dst File",
			expectFile: true,
		},
		{
			name:       "srcFileGlob",
			src:        srcFileGlob,
			dst:        "dstDir/",
			expectPath: "dstDir/srcFile",
			expectFile: true,
		},
		{
			name:       "srcFileGlobNoDest",
			src:        srcFileGlob,
			dst:        "",
			expectPath: srcFile,
			expectFile: true,
		},
		{
			name:       "srcFileNestedGlob",
			src:        srcFileNestedGlob,
			dst:        "dstDir/",
			expectPath: "dstDir/srcFileNested",
			expectFile: true,
		},
		{
			name:       "srcFileNestedGlobNoDest",
			src:        srcFileNestedGlob,
			dst:        "",
			expectPath: srcFileNested,
			expectFile: true,
		},
		{
			name: "dstRestricted",
			src:  srcFile,
			// Will be restricted to `/` in the rootfs and should copy to there OK
			dst:        "../../../../",
			expectPath: "srcFile",
			expectFile: true,
		},
		// Source is a Directory
		{
			name:       "SrcDirNoDest",
			src:        srcDir,
			dst:        "",
			expectPath: srcDir,
			expectDir:  true,
		},
		{
			name:       "SrcDirDest",
			src:        srcDir,
			dst:        "dstDir",
			expectPath: "dstDir",
			expectDir:  true,
		},
		{
			name:       "SrcDirToDir",
			src:        srcDir,
			dst:        "dstDir/",
			expectPath: "dstDir/srcDir",
			expectDir:  true,
		},
		{
			name:       "srcDirToDirLongPath",
			src:        srcDir,
			dst:        "dstDir/long/path/to/srcDir",
			expectPath: "dstDir/long/path/to/srcDir",
			expectDir:  true,
		},
		{
			name:       "srcDirSpace",
			src:        srcSpaceDir,
			dst:        "src Dir",
			expectPath: "src Dir",
			expectDir:  true,
		},
		{
			name:       "dstDirSpace",
			src:        srcDir,
			dst:        "dst Dir",
			expectPath: "dst Dir",
			expectDir:  true,
		},
		{
			name:       "srcDirGlob",
			src:        srcDirGlob,
			dst:        "dstDir/",
			expectPath: "dstDir/srcDir",
			expectDir:  true,
		},
		{
			name:       "srcDirGlobNoDest",
			src:        srcDirGlob,
			dst:        "",
			expectPath: srcDir,
			expectDir:  true,
		},
		// Source is a Symlink
		{
			name:       "srcFileLinkRel",
			src:        srcFileLinkRel,
			dst:        "srcFileLinkRel",
			expectPath: "srcFileLinkRel",
			// Copied the file, not the link itself
			expectFile: true,
		},
		{
			name:       "srcFileLinkAbs",
			src:        srcFileLinkAbs,
			dst:        "srcFileLinkAbs",
			expectPath: "srcFileLinkAbs",
			// Copied the file, not the link itself
			expectFile: true,
		},
		{
			name:       "srcDirLinkRel",
			src:        srcDirLinkRel,
			dst:        "srcDirLinkRel",
			expectPath: "srcDirLinkRel",
			// Copied the dir, not the link itself
			expectDir: true,
		},
		{
			name:       "srcDirLinkAbs",
			src:        srcDirLinkAbs,
			dst:        "srcDirLinkAbs",
			expectPath: "srcDirLinkAbs",
			// Copied the dir, not the link itself
			expectDir: true,
		},
		// issue 261 - multiple globbed sources, with no dest
		// both srcfile and srcdir should be copied for glob of "src*"
		{
			name:       "srcDirGlobNoDestMulti1",
			src:        srcGlob,
			dst:        "",
			expectPath: srcDir,
			expectDir:  true,
		},
		{
			name:       "srcDirGlobNoDestMulti2",
			src:        srcGlob,
			dst:        "",
			expectPath: srcFile,
			expectFile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create outer destination dir
			dstRoot, err := ioutil.TempDir("", "copy-test-dst-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(dstRoot)

			if err := CopyFromHost(tt.src, tt.dst, dstRoot); err != nil {
				t.Errorf("unexpected failure running %s test: %s", t.Name(), err)
			}

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
				t.Errorf("destination should be a file, but isn't")
			}
			// Dir when expected?
			if tt.expectDir && !fs.IsDir(dstFinal) {
				t.Errorf("destination should be a directory, but isn't")
			}
			// None of these test cases should result in dst being a symlink
			if fs.IsLink(dstFinal) {
				t.Errorf("destination should not be a symlink, but is")
			}
		})
	}
}

// TestCopyFromHostNested tests that copying a single directory containing nested dirs, files, links
// works. CopyFromHost should always resolve symlinks, even those nested inside a source dir.
func TestCopyFromHostNested(t *testing.T) {
	// create tmpdir
	dir, err := ioutil.TempDir("", "copy-test-src-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// All our test files/dirs/links will be nested inside innerDir
	innerDir := filepath.Join(dir, "innerDir")
	if err := os.Mkdir(innerDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Source Files
	srcFile := filepath.Join(innerDir, "srcFile")
	if err := ioutil.WriteFile(srcFile, []byte(sourceFileContent), 0644); err != nil {
		t.Fatal(err)
	}
	// Source Dirs
	srcDir := filepath.Join(innerDir, "srcDir")
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Source Symlinks
	srcFileLinkAbs := filepath.Join(innerDir, "srcFileLinkAbs")
	if err := os.Symlink(srcFile, srcFileLinkAbs); err != nil {
		t.Fatal(err)
	}
	srcFileLinkRel := filepath.Join(innerDir, "srcFileLinkRel")
	if err := os.Symlink("./srcFile", srcFileLinkRel); err != nil {
		t.Fatal(err)
	}
	srcDirLinkAbs := filepath.Join(innerDir, "srcDirLinkAbs")
	if err := os.Symlink(srcDir, srcDirLinkAbs); err != nil {
		t.Fatal(err)
	}
	srcDirLinkRel := filepath.Join(innerDir, "srcDirLinkRel")
	if err := os.Symlink("./srcDir", srcDirLinkRel); err != nil {
		t.Fatal(err)
	}

	// Create outer destination dir
	dstDir, err := ioutil.TempDir("", "copy-test-dst-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dstDir)

	// Copy our source innerDir over into the destination dir
	if err := CopyFromHost(innerDir, "innerDir", dstDir); err != nil {
		t.Errorf("unexpected failure copying directory: %s", err)
	}

	// Now verify all the nested copied files are as expected
	tests := []struct {
		expectPath string
		expectFile bool
		expectDir  bool
	}{
		{
			expectPath: "innerDir/srcFile",
			expectFile: true,
		},
		{
			expectPath: "innerDir/srcDir",
			expectDir:  true,
		},
		// Source is a Symlink
		// Should always have copied the target, not the link.
		{
			expectPath: "innerDir/srcFileLinkRel",
			expectFile: true,
		},
		{
			expectPath: "innerDir/srcFileLinkAbs",
			expectFile: true,
		},
		{
			expectPath: "innerDir/srcDirLinkRel",
			expectDir:  true,
		},
		{
			expectPath: "innerDir/srcDirLinkAbs",
			expectDir:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.expectPath, func(t *testing.T) {
			dstFinal := filepath.Join(dstDir, tt.expectPath)
			// File when expected?
			if tt.expectFile && !fs.IsFile(dstFinal) {
				t.Errorf("destination should be a file, but isn't")
			}
			// Dir when expected?
			if tt.expectDir && !fs.IsDir(dstFinal) {
				t.Errorf("destination should be a directory, but isn't")
			}
			// None of these test cases should result in dst being a symlink
			if fs.IsLink(dstFinal) {
				t.Errorf("destination should not be a symlink, but is")
			}
		})
	}

}

// TestCopyFromStage tests that copying non-nested source dirs, files, links to various
// destinations works. CopyFromStage should resolve top-level symlinks for sources it is
// called against.
func TestCopyFromStage(t *testing.T) {
	// create tmpdir
	srcRoot, err := ioutil.TempDir("", "copy-test-src-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(srcRoot)

	// Source Files
	srcFile := filepath.Join(srcRoot, "srcFile")
	if err := ioutil.WriteFile(srcFile, []byte(sourceFileContent), 0644); err != nil {
		t.Fatal(err)
	}
	srcSpaceFile := filepath.Join(srcRoot, "src File")
	if err := ioutil.WriteFile(srcSpaceFile, []byte(sourceFileContent), 0644); err != nil {
		t.Fatal(err)
	}
	// Source Dirs
	srcDir := filepath.Join(srcRoot, "srcDir")
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcSpaceDir := filepath.Join(srcRoot, "src Dir")
	if err := os.Mkdir(srcSpaceDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Nested File (to test multi level glob)
	srcFileNested := filepath.Join(srcRoot, "srcDir/srcFileNested")
	if err := ioutil.WriteFile(srcFileNested, []byte(sourceFileContent), 0644); err != nil {
		t.Fatal(err)
	}
	// Source Symlinks
	// Note the absolute links are absolute paths inside the srcRoot
	srcFileLinkAbs := filepath.Join(srcRoot, "srcFileLinkAbs")
	if err := os.Symlink("/srcFile", srcFileLinkAbs); err != nil {
		t.Fatal(err)
	}
	srcFileLinkRel := filepath.Join(srcRoot, "srcFileLinkRel")
	if err := os.Symlink("./srcFile", srcFileLinkRel); err != nil {
		t.Fatal(err)
	}
	srcDirLinkAbs := filepath.Join(srcRoot, "srcDirLinkAbs")
	if err := os.Symlink("/srcDir", srcDirLinkAbs); err != nil {
		t.Fatal(err)
	}
	srcDirLinkRel := filepath.Join(srcRoot, "srcDirLinkRel")
	if err := os.Symlink("./srcDir", srcDirLinkRel); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		srcRel     string
		dstRel     string
		expectPath string
		expectFile bool
		expectDir  bool
	}{
		// Source is a file
		{
			name:       "SrcFileNoDest",
			srcRel:     "srcFile",
			dstRel:     "",
			expectPath: "srcFile",
			expectFile: true,
		},
		{
			name:       "SrcFileAbs",
			srcRel:     "/srcFile",
			dstRel:     "",
			expectPath: "srcFile",
			expectFile: true,
		},
		{
			name:       "SrcFileToDir",
			srcRel:     "srcFile",
			dstRel:     "dstDir/",
			expectPath: "dstDir/srcFile",
			expectFile: true,
		},
		{
			name:       "srcFileToFile",
			srcRel:     "srcFile",
			dstRel:     "dstDir/dstFile",
			expectPath: "dstDir/dstFile",
			expectFile: true,
		},
		{
			name:       "srcFileToFileLongPath",
			srcRel:     "srcFile",
			dstRel:     "dstDir/long/path/to/dstFile",
			expectPath: "dstDir/long/path/to/dstFile",
			expectFile: true,
		},
		{
			name:       "srcFileSpace",
			srcRel:     "src File",
			dstRel:     "",
			expectPath: "src File",
			expectFile: true,
		},
		{
			name:       "dstFileSpace",
			srcRel:     "srcFile",
			dstRel:     "dst File",
			expectPath: "dst File",
			expectFile: true,
		},
		{
			name:       "srcFileGlob",
			srcRel:     "srcF?*",
			dstRel:     "dstDir/",
			expectPath: "dstDir/srcFile",
			expectFile: true,
		},
		{
			name:       "srcFileGlobNoDest",
			srcRel:     "srcF?*",
			dstRel:     "",
			expectPath: "srcFile",
			expectFile: true,
		},
		{
			name:       "srcFileNestedGlob",
			srcRel:     "srcDi?/srcFil?Nested",
			dstRel:     "dstDir/",
			expectPath: "dstDir/srcFileNested",
			expectFile: true,
		},
		{
			name:       "srcFileNestedGlobNoDest",
			srcRel:     "srcDi?/srcFil?Nested",
			dstRel:     "",
			expectPath: "srcDir/srcFileNested",
			expectFile: true,
		},
		{
			name:   "dstRestricted",
			srcRel: "srcFile",
			// Will be restricted to `/` in the dst rootfs and should copy to there OK
			dstRel:     "../../../../",
			expectPath: "srcFile",
			expectFile: true,
		},
		{
			name: "srcRestricted",
			// Will be restricted to `/srcFile` in the src rootfs and should copy from there OK
			srcRel:     "../../../../srcFile",
			dstRel:     "",
			expectPath: "srcFile",
			expectFile: true,
		},
		// Source is a Directory
		{
			name:       "SrcDirNoDest",
			srcRel:     "srcDir",
			dstRel:     "",
			expectPath: "srcDir",
			expectDir:  true,
		},
		{
			name:       "SrcDirDest",
			srcRel:     "srcDir",
			dstRel:     "dstDir",
			expectPath: "dstDir",
			expectDir:  true,
		},
		{
			name:       "SrcDirToDir",
			srcRel:     "srcDir",
			dstRel:     "dstDir/",
			expectPath: "dstDir/srcDir",
			expectDir:  true,
		},
		{
			name:       "srcDirToDirLongPath",
			srcRel:     "srcDir",
			dstRel:     "dstDir/long/path/to/srcDir",
			expectPath: "dstDir/long/path/to/srcDir",
			expectDir:  true,
		},
		{
			name:       "srcDirSpace",
			srcRel:     "src Dir",
			dstRel:     "",
			expectPath: "src Dir",
			expectDir:  true,
		},
		{
			name:       "dstDirSpace",
			srcRel:     "srcDir",
			dstRel:     "dst Dir",
			expectPath: "dst Dir",
			expectDir:  true,
		},
		{
			name:       "srcDirGlob",
			srcRel:     "srcD?*",
			dstRel:     "dstDir/",
			expectPath: "dstDir/srcDir",
			expectDir:  true,
		},
		{
			name:       "srcDirGlobNoDest",
			srcRel:     "srcD?*",
			dstRel:     "",
			expectPath: "srcDir",
			expectDir:  true,
		},
		// Source is a Symlink
		{
			name:       "srcFileLinkRel",
			srcRel:     "srcFileLinkRel",
			dstRel:     "",
			expectPath: "srcFileLinkRel",
			// Copied the file, not the link itself
			expectFile: true,
		},
		{
			name:       "srcFileLinkAbs",
			srcRel:     "srcFileLinkAbs",
			dstRel:     "",
			expectPath: "srcFileLinkAbs",
			// Copied the file, not the link itself
			expectFile: true,
		},
		{
			name:       "srcDirLinkRel",
			srcRel:     "srcDirLinkRel",
			dstRel:     "",
			expectPath: "srcDirLinkRel",
			// Copied the dir, not the link itself
			expectDir: true,
		},
		{
			name:       "srcDirLinkAbs",
			srcRel:     "srcDirLinkAbs",
			dstRel:     "",
			expectPath: "srcDirLinkAbs",
			// Copied the dir, not the link itself
			expectDir: true,
		},
		// issue 261 - multiple globbed sources, with no dest
		// both srcfile and srcdir should be copied for glob of "src*"
		{
			name:       "srcDirGlobNoDestMulti1",
			srcRel:     "src*",
			dstRel:     "",
			expectPath: "srcDir",
			expectDir:  true,
		},
		{
			name:       "srcDirGlobNoDestMulti2",
			srcRel:     "src*",
			dstRel:     "",
			expectPath: "srcFile",
			expectFile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create outer destination dir
			dstRoot, err := ioutil.TempDir("", "copy-test-dst-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(dstRoot)

			// Manually concatenating because we need to preserve any trailing slash that is
			// stripped by Join.
			if err := CopyFromStage(tt.srcRel, tt.dstRel, srcRoot, dstRoot); err != nil {
				t.Errorf("unexpected failure running %s test: %s", t.Name(), err)
			}

			dstFinal := filepath.Join(dstRoot, tt.expectPath)
			// verify file was copied
			_, err = os.Stat(dstFinal)
			if err != nil && !os.IsNotExist(err) {
				t.Fatalf("while checking for destination file: %s", err)
			}
			if os.IsNotExist(err) {
				t.Errorf("expected destination %s does not exist", tt.expectPath)
			}

			// File when expected?
			if tt.expectFile && !fs.IsFile(dstFinal) {
				t.Errorf("destination should be a file, but isn't")
			}
			// Dir when expected?
			if tt.expectDir && !fs.IsDir(dstFinal) {
				t.Errorf("destination should be a directory, but isn't")
			}
			// None of these test cases should result in dst being a symlink
			if fs.IsLink(dstFinal) {
				t.Errorf("destination should not be a symlink, but is")
			}
		})
	}
}

// TestCopyFromStageNested tests that copying a single directory containing nested dirs, files, links
// works. CopyFromStage should *not* resolve the symlinks that are nested in the dir.
func TestCopyFromStageNested(t *testing.T) {
	// create tmpdir
	srcRoot, err := ioutil.TempDir("", "copy-test-src-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(srcRoot)

	// All our test files/dirs/links will be nested inside innerDir
	innerDir := filepath.Join(srcRoot, "innerDir")
	if err := os.Mkdir(innerDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Source Files
	srcFile := filepath.Join(innerDir, "srcFile")
	if err := ioutil.WriteFile(srcFile, []byte(sourceFileContent), 0644); err != nil {
		t.Fatal(err)
	}
	// Source Dirs
	srcDir := filepath.Join(innerDir, "srcDir")
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Source Symlinks
	srcFileLinkAbs := filepath.Join(innerDir, "srcFileLinkAbs")
	if err := os.Symlink(srcFile, srcFileLinkAbs); err != nil {
		t.Fatal(err)
	}
	srcFileLinkRel := filepath.Join(innerDir, "srcFileLinkRel")
	if err := os.Symlink("./srcFile", srcFileLinkRel); err != nil {
		t.Fatal(err)
	}
	srcDirLinkAbs := filepath.Join(innerDir, "srcDirLinkAbs")
	if err := os.Symlink(srcDir, srcDirLinkAbs); err != nil {
		t.Fatal(err)
	}
	srcDirLinkRel := filepath.Join(innerDir, "srcDirLinkRel")
	if err := os.Symlink("./srcDir", srcDirLinkRel); err != nil {
		t.Fatal(err)
	}

	// Create outer destination dir
	dstRoot, err := ioutil.TempDir("", "copy-test-dst-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dstRoot)

	// Copy our source innerDir over into the destination dir
	if err := CopyFromStage("innerDir", "", srcRoot, dstRoot); err != nil {
		t.Errorf("unexpected failure copying directory: %s", err)
	}

	// Now verify all the nested copied files are as expected
	tests := []struct {
		expectPath string
		expectFile bool
		expectDir  bool
		expectLink bool
	}{
		{
			expectPath: "innerDir/srcFile",
			expectFile: true,
		},
		{
			expectPath: "innerDir/srcDir",
			expectDir:  true,
		},
		// Nested symlink, inside the src directory.
		// Should always have copied the link itself.
		{
			expectPath: "innerDir/srcFileLinkRel",
			expectFile: true,
			expectLink: true,
		},
		{
			expectPath: "innerDir/srcFileLinkAbs",
			expectFile: true,
			expectLink: true,
		},
		{
			expectPath: "innerDir/srcDirLinkRel",
			expectDir:  true,
			expectLink: true,
		},
		{
			expectPath: "innerDir/srcDirLinkAbs",
			expectDir:  true,
			expectLink: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.expectPath, func(t *testing.T) {
			dstFinal := filepath.Join(dstRoot, tt.expectPath)
			// File when expected?
			if tt.expectFile && !fs.IsFile(dstFinal) {
				t.Errorf("destination should be a file, but isn't")
			}
			// Dir when expected?
			if tt.expectDir && !fs.IsDir(dstFinal) {
				t.Errorf("destination should be a directory, but isn't")
			}
			// Link when expected?
			if tt.expectLink && !fs.IsLink(dstFinal) {
				t.Errorf("destination should be a symlink, but isn't")
			}
			// Not a link when not expected
			if !tt.expectLink && fs.IsLink(dstFinal) {
				t.Errorf("destination should not be a symlink, but is")
			}
		})
	}

}

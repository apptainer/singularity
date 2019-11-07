// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package fs

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestEnsureFileWithPermission(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tmpDir, err := ioutil.TempDir("", "ensure_file_perm-")
	if err != nil {
		t.Errorf("Unable to make tmpdir %s", err)
	}

	//
	// First test: Ensure a already-existing file is the
	// correct permssion.
	//

	existFile := filepath.Join(tmpDir, "already-exists")

	// Create the test file.
	fp, err := os.OpenFile(existFile, os.O_CREATE, 0755)
	if err != nil {
		t.Errorf("Unable to create test file: %s", err)
	}

	// Ensure the test file is the currect permission.
	err = fp.Chmod(0755)
	if err != nil {
		t.Errorf("Unable to change file permission: %s", err)
	}

	// Check the permissions.
	finfo, err := fp.Stat()
	if err != nil {
		t.Errorf("Unable to stat file: %s", err)
	}

	// Double check the permission is what we expect.
	if currentMode := finfo.Mode(); currentMode != 0755 {
		t.Errorf("Unexpect file permission: expecting 755, got %o", currentMode)
	}

	// Now the actral test!
	err = EnsureFileWithPermission(existFile, 0655)
	if err != nil {
		t.Errorf("Failed to ensure file permission: %s", err)
	}

	// Re-stat the file.
	finfo, err = fp.Stat()
	if err != nil {
		t.Errorf("Unable to stat file: %s", err)
	}

	// Finally, check the file permission.
	if currentMode := finfo.Mode(); currentMode != 0655 {
		t.Errorf("Unexpect file permission: expecting 655, got %o", currentMode)
	}

	// Test again with another permission.
	err = EnsureFileWithPermission(existFile, 0777)
	if err != nil {
		t.Errorf("Failed to ensure file permission: %s", err)
	}

	// Re-stat the file.
	finfo, err = fp.Stat()
	if err != nil {
		t.Errorf("Unable to stat file: %s", err)
	}

	// Finally, check the file permission.
	if currentMode := finfo.Mode(); currentMode != 0777 {
		t.Errorf("Unexpect file permission: expecting 777, got %o", currentMode)
	}

	// And close the file.
	fp.Close()

	//
	// Second test: Ensure a non-existing file is the
	// correct permssion.
	//

	nonExistFile := filepath.Join(tmpDir, "non-exists")

	// This test, EnsureFileWithPermission will need to create
	// this file, with the correct permission.
	err = EnsureFileWithPermission(nonExistFile, 0755)
	if err != nil {
		t.Errorf("Failed to create/ensure file permission: %s", err)
	}

	// Stat the file.
	einfo, err := os.Stat(nonExistFile)
	if err != nil {
		t.Errorf("Unable to stat file: %s", err)
	}

	// Finally, check the file permission.
	if currentMode := einfo.Mode(); currentMode != 0755 {
		t.Errorf("Unexpect file permission: expecting 755, got %o", currentMode)
	}

	// Test again with another permission.
	err = EnsureFileWithPermission(nonExistFile, 0544)
	if err != nil {
		t.Errorf("Failed to ensure file permission: %s", err)
	}

	// Stat the file again.
	einfo, err = os.Stat(nonExistFile)
	if err != nil {
		t.Errorf("Unable to stat file: %s", err)
	}

	// Finally, check the file permission.
	if currentMode := einfo.Mode(); currentMode != 0544 {
		t.Errorf("Unexpect file permission: expecting 544, got %o", currentMode)
	}

	// Cleanup.
	err = os.RemoveAll(tmpDir)
	if err != nil {
		t.Errorf("Unable to remove tmpdir: %s", err)
	}
}

func TestIsFile(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	if IsFile("/etc/passwd") != true {
		t.Errorf("IsFile returns false for file")
	}
}

func TestIsDir(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	if IsDir("/etc") != true {
		t.Errorf("IsDir returns false for directory")
	}
}

func TestIsLink(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	if IsLink("/proc/mounts") != true {
		t.Errorf("IsLink returns false for link")
	}
}

func TestIsOwner(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	if IsOwner("/etc/passwd", 0) != true {
		t.Errorf("IsOwner returns false for /etc/passwd owner")
	}
}

func TestIsExec(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	if IsExec("/bin/ls") != true {
		t.Errorf("IsExec returns false for /bin/ls execution bit")
	}
}

func TestIsSuid(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	if IsSuid("/bin/su") != true {
		t.Errorf("IsSuid returns false for /bin/su setuid bit")
	}
}

func TestRootDir(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	var tests = []struct {
		path     string
		expected string
	}{
		{"/path/to/something", "/path"},
		{"relative/path", "relative"},
		{"/path/to/something/", "/path"},
		{"relative/path/", "relative"},
		{"/path", "/path"},
		{"/path/", "/path"},
		{"/path/../something", "/something"},
		{"/", "/"},
		{"./", "."},
		{"/.././", "/"},
		{"./path", "path"},
		{"../path", ".."},
		{"", "."},
	}

	for _, test := range tests {
		if rootpath := RootDir(test.path); rootpath != test.expected {
			t.Errorf("RootDir(%v) != \"%v\" (function returned %v)", test.path, test.expected, rootpath)
		}
	}
}

func TestMkdirAll(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tmpdir, err := ioutil.TempDir("", "mkdir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	if err := MkdirAll(filepath.Join(tmpdir, "test"), 0777); err != nil {
		t.Error(err)
	}
	if err := MkdirAll(filepath.Join(tmpdir, "test/test"), 0000); err != nil {
		t.Error(err)
	}
	if err := MkdirAll(filepath.Join(tmpdir, "test/test/test"), 0755); err == nil {
		t.Errorf("should have failed with a permission denied")
	}
	fi, err := os.Stat(filepath.Join(tmpdir, "test"))
	if err != nil {
		t.Error(err)
	}
	if fi.Mode().Perm() != 0777 {
		t.Errorf("bad mode applied on %s, got %v", filepath.Join(tmpdir, "test"), fi.Mode().Perm())
	}
}

func TestMkdir(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tmpdir, err := ioutil.TempDir("", "mkdir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	test := filepath.Join(tmpdir, "test")
	if err := Mkdir(test, 0777); err != nil {
		t.Error(err)
	}
	fi, err := os.Stat(test)
	if err != nil {
		t.Error(err)
	}
	if fi.Mode().Perm() != 0777 {
		t.Errorf("bad mode applied on %s, got %v", test, fi.Mode().Perm())
	}
}

func TestEvalRelative(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tmpdir, err := ioutil.TempDir("", "evalrelative")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	// test layout
	// - /bin -> usr/bin
	// - /sbin -> usr/sbin
	// - /usr/bin
	// - /usr/sbin
	// - /bin/bin -> /bin
	// - /bin/sbin -> ../sbin
	// - /sbin/sbin2 -> ../../sbin

	// symlink $TMP/bin -> usr/bin
	os.Symlink("usr/bin", filepath.Join(tmpdir, "bin"))
	// symlink $TMP/sbin -> usr/sbin
	os.Symlink("usr/sbin", filepath.Join(tmpdir, "sbin"))

	// directory $TMP/usr/bin
	MkdirAll(filepath.Join(tmpdir, "usr", "bin"), 0755)
	// directory $TMP/usr/sbin
	MkdirAll(filepath.Join(tmpdir, "usr", "sbin"), 0755)

	// symlink $TMP/usr/bin/bin -> /bin
	os.Symlink("/bin", filepath.Join(tmpdir, "bin", "bin"))
	// symlink $TMP/usr/bin/sbin -> ../sbin
	os.Symlink("../sbin", filepath.Join(tmpdir, "bin", "sbin"))
	// symlink $TMP/usr/sbin/sbin2 -> ../../sbin
	os.Symlink("../../sbin", filepath.Join(tmpdir, "sbin", "sbin2"))
	// symlink $TMP/rootfs -> ../../../../
	os.Symlink("../../../../", filepath.Join(tmpdir, "rootfs"))

	// symlink $TMP/pool -> loop
	os.Symlink("loop", filepath.Join(tmpdir, "pool"))
	// symlink $TMP/loop -> pool
	os.Symlink("pool", filepath.Join(tmpdir, "loop"))
	// symlink $TMP/loop2 -> loop2
	os.Symlink("loop2", filepath.Join(tmpdir, "loop2"))

	testPath := []struct {
		path string
		eval string
	}{
		{"", "/"},
		{"/", "/"},
		{"/bin", "/usr/bin"},
		{"/sbin", "/usr/sbin"},
		{"/bin/bin", "/usr/bin"},
		{"/bin/sbin", "/usr/sbin"},
		{"/sbin/sbin2", "/usr/sbin"},
		{"/bin/test", "/usr/bin/test"},
		{"/sbin/test", "/usr/sbin/test"},
		{"/usr/bin/test", "/usr/bin/test"},
		{"/usr/sbin/test", "/usr/sbin/test"},
		{"/bin/bin/test", "/usr/bin/test"},
		{"/bin/sbin/test", "/usr/sbin/test"},
		{"/sbin/sbin2/test", "/usr/sbin/test"},
		{"/bin/bin/sbin/sbin2/test", "/usr/sbin/test"},
		{"/rootfs", "/"},
		{"/fake/test", "/fake/test"},
		{"/loop", "/loop"},
		{"/loop2", "/loop2"},
	}

	for _, p := range testPath {
		// evaluate paths from $TMP test directory
		eval := EvalRelative(p.path, tmpdir)
		if eval != p.eval {
			t.Errorf("evaluated path %s expected path %s got %s", p.path, p.eval, eval)
		}
	}
}

func TestTouch(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tmpdir, err := ioutil.TempDir("", "evalrelative")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	if err := Touch(tmpdir); err == nil {
		t.Errorf("touch can't take a directory")
	}

	testing := filepath.Join(tmpdir, "testing")

	if err := Touch(testing); err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(testing); os.IsNotExist(err) {
		t.Errorf("creation of %s failed", testing)
	}
}

func TestMakeTempDir(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name          string
		basedir       string
		pattern       string
		mode          os.FileMode
		expectSuccess bool
	}{
		{"tmp directory with 0700", "", "atmp-", 0700, true},
		{"tmp directory with 0755", "", "btmp-", 0755, true},
		{"root directory 0700", "/", "bad-", 0700, false},
		{"with non-existent basedir", "/tmp/__utest__", "ctmp-", 0700, false},
		{"with existent basedir", "/var/tmp", "dtmp-", 0700, true},
	}
	for _, tt := range tests {
		d, err := MakeTmpDir(tt.basedir, tt.pattern, tt.mode)
		if err != nil && tt.expectSuccess {
			t.Errorf("%s: unexpected failure: %s", tt.name, err)
		} else if err == nil && !tt.expectSuccess {
			t.Errorf("%s: unexpected success", tt.name)
		} else if err != nil && !tt.expectSuccess {
			// no check, fail as expected
			continue
		}
		defer os.Remove(d)

		fi, err := os.Stat(d)
		if err != nil {
			t.Fatalf("%s: could not stat %s: %s", tt.name, d, err)
		}
		expectedMode := tt.mode | os.ModeDir
		if fi.Mode() != expectedMode {
			t.Fatalf(
				"%s: unexpected mode returned for directory %s, %o instead of %o",
				tt.name, d, fi.Mode(), expectedMode,
			)
		}
		expectedPrefix := filepath.Join(os.TempDir(), tt.pattern)
		if tt.basedir != "" {
			expectedPrefix = filepath.Join(tt.basedir, tt.pattern)
		}
		if !strings.HasPrefix(d, expectedPrefix) {
			t.Fatalf("%s: unexpected prefix returned in path %s, expected %s", tt.name, d, expectedPrefix)
		}
	}
}

func TestMakeTempFile(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name          string
		basedir       string
		pattern       string
		mode          os.FileMode
		expectSuccess bool
	}{
		{"tmp file with 0700", "", "atmp-", 0700, true},
		{"tmp file with 0755", "", "btmp-", 0755, true},
		{"root directory tmp file 0700", "/", "bad-", 0700, false},
		{"with non-existent basedir", "/tmp/__utest__", "ctmp-", 0700, false},
		{"with existent basedir", "/var/tmp", "dtmp-", 0700, true},
	}
	for _, tt := range tests {
		f, err := MakeTmpFile(tt.basedir, tt.pattern, tt.mode)
		if err != nil && tt.expectSuccess {
			t.Errorf("%s: unexpected failure: %s", tt.name, err)
		} else if err == nil && !tt.expectSuccess {
			t.Errorf("%s: unexpected success", tt.name)
		} else if err != nil && !tt.expectSuccess {
			// no check, fail as expected
			continue
		}
		defer f.Close()
		defer os.Remove(f.Name())

		fileName := f.Name()

		fi, err := f.Stat()
		if err != nil {
			t.Fatalf("%s: could not stat %s: %s", tt.name, fileName, err)
		}
		expectedMode := tt.mode
		if fi.Mode() != expectedMode {
			t.Fatalf(
				"%s: unexpected mode returned for file %s, %o instead of %o",
				tt.name, fileName, fi.Mode(), expectedMode,
			)
		}
		expectedPrefix := filepath.Join(os.TempDir(), tt.pattern)
		if tt.basedir != "" {
			expectedPrefix = filepath.Join(tt.basedir, tt.pattern)
		}
		if !strings.HasPrefix(fileName, expectedPrefix) {
			t.Fatalf("%s: unexpected prefix returned in path %s, expected %s", tt.name, fileName, expectedPrefix)
		}
	}
}

func TestCopyFile(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	testData := []byte("Hello, Singularity!")

	tmpDir, err := ioutil.TempDir("", "copy-file")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	source := filepath.Join(tmpDir, "source")
	err = ioutil.WriteFile(source, testData, 0644)
	if err != nil {
		t.Fatalf("failed to create test source file: %v", err)
	}

	tt := []struct {
		name        string
		from        string
		to          string
		mode        os.FileMode
		expectError string
	}{
		{
			name:        "non existent source",
			from:        filepath.Join(tmpDir, "not-there"),
			to:          filepath.Join(tmpDir, "invalid"),
			mode:        0644,
			expectError: "no such file or directory",
		},
		{
			name:        "non existent target",
			from:        source,
			to:          filepath.Join(os.TempDir(), "not-there", "invalid"),
			mode:        0644,
			expectError: "no such file or directory",
		},
		{
			name: "change mode",
			from: source,
			to:   filepath.Join(tmpDir, "executable"),
			mode: 0755,
		},
		{
			name: "simple copy",
			from: source,
			to:   filepath.Join(tmpDir, "copy"),
			mode: 0644,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := CopyFile(tc.from, tc.to, tc.mode)
			if tc.expectError == "" && err != nil {
				t.Fatalf("expected no error, but got %v", err)
			}
			if tc.expectError != "" && err == nil {
				t.Fatalf("expected error, but got nil")
			}
			if err != nil && !strings.Contains(err.Error(), tc.expectError) {
				t.Fatalf("expected error to contain %q, but got %q", tc.expectError, err)
			}

			if tc.expectError == "" {
				actual, err := ioutil.ReadFile(tc.to)
				if err != nil {
					t.Fatalf("could not read copied file: %v", err)
				}
				if !bytes.Equal(actual, testData) {
					t.Fatalf("copied content mismatch")
				}
				fi, err := os.Stat(tc.to)
				if err != nil {
					t.Fatalf("could not read copied file info")
				}
				if fi.Mode() != tc.mode {
					t.Fatalf("expected %s mode, but gor %s", tc.mode, fi.Mode())
				}
			}
		})
	}
}

func TestIsWritable(t *testing.T) {
	test.EnsurePrivilege(t)

	// Directories owned by root, we will check later if the unprivileged user can access it.
	validRoot755Dir, err := MakeTmpDir("", "", 0755)
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s: %s", validRoot755Dir, err)
	}
	defer os.RemoveAll(validRoot755Dir)

	validRoot777Dir, err := MakeTmpDir("", "", 0777)
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s: %s", validRoot777Dir, err)
	}
	defer os.RemoveAll(validRoot777Dir)

	// Fall back under the unprivileged user.
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// We make a temporary directory where all the different cases will be tested.
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}
	defer os.RemoveAll(tempDir)

	// All the directory that we are about to create will be deleted when the temporary
	// directory will be removed.
	validWritablePath := filepath.Join(tempDir, "writableDir")
	validNotWritablePath := filepath.Join(tempDir, "notWritableDir")
	valid700Dir := filepath.Join(tempDir, "700Dir")
	valid555Dir := filepath.Join(tempDir, "555Dir")
	err = os.MkdirAll(validWritablePath, 0755)
	if err != nil {
		t.Fatalf("failed to create directory %s: %s", validWritablePath, err)
	}
	err = os.MkdirAll(validNotWritablePath, 0444)
	if err != nil {
		t.Fatalf("failed to create directory %s: %s", validNotWritablePath, err)
	}
	err = os.MkdirAll(valid700Dir, 0700)
	if err != nil {
		t.Fatalf("failed to create directory %s: %s", valid700Dir, err)
	}
	err = os.MkdirAll(valid555Dir, 0555)
	if err != nil {
		t.Fatalf("failed to create directory %s: %s", valid555Dir, err)
	}

	tests := []struct {
		name           string
		path           string
		expectedResult bool
	}{
		{
			name:           "empty path",
			path:           "",
			expectedResult: false,
		},
		{
			name:           "writable path",
			path:           validWritablePath,
			expectedResult: true,
		},
		{
			name:           "700 directory",
			path:           valid700Dir,
			expectedResult: true,
		},
		{
			name:           "555 directory",
			path:           valid555Dir,
			expectedResult: false,
		},
		{
			name:           "root-owned 755 directory",
			path:           validRoot755Dir,
			expectedResult: false,
		},
		{
			name:           "root-owned 777 directory",
			path:           validRoot777Dir,
			expectedResult: true,
		},
		{
			name:           "not writable path",
			path:           validNotWritablePath,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test.DropPrivilege(t)
			writable := IsWritable(tt.path)
			if tt.expectedResult != writable {
				t.Fatalf("test %s returned %v instead of %v (%s)", tt.name, writable, tt.expectedResult, tt.path)
			}
		})
	}

}

func TestFirstExistingParent(t *testing.T) {
	testDir, err := MakeTmpDir("", "dir", 0755)
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s: %s", testDir, err)
	}
	defer os.RemoveAll(testDir)

	testFile, err := MakeTmpFile(testDir, "file", 0644)
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s: %s", testFile.Name(), err)
	}
	testFile.Close()
	defer os.RemoveAll(testFile.Name())

	tests := []struct {
		name    string
		path    string
		correct string
	}{
		{
			name:    "path exists",
			path:    testFile.Name(),
			correct: testFile.Name(),
		},
		{
			name:    "path missing file",
			path:    filepath.Join(testDir, "notafile"),
			correct: testDir,
		},
		{
			name:    "path missing dir",
			path:    filepath.Join(os.TempDir(), "notadir", "notafile"),
			correct: os.TempDir(),
		},
		{
			name:    "root is first parent",
			path:    filepath.Join("/", "notadir", "notafile"),
			correct: "/",
		},
		{
			name:    "root dir",
			path:    "/",
			correct: "/",
		},
		{
			name:    "cwd",
			path:    ".",
			correct: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test.DropPrivilege(t)
			path, err := FirstExistingParent(tt.path)
			if err != nil {
				t.Errorf("unexpected error finding first existing partent for path %q: %v", tt.path, err)
			}
			if tt.correct != path {
				t.Errorf("test %s returned %v instead of %v (%s)", tt.name, path, tt.correct, tt.path)
			}
		})
	}
}

func TestForceRemoveAll(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)
	// Setup a structure that os.RemoveAll should fail to remove
	testDir, err := MakeTmpDir("", "dir", 0755)
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s: %s", testDir, err)
	}
	testFile, err := MakeTmpFile(testDir, "file", 0644)
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s: %s", testFile.Name(), err)
	}
	testFile.Close()
	// Change the perm on testDir so that RemoveAll should fail
	err = os.Chmod(testDir, 000)
	if err != nil {
		t.Fatalf("failed to set permissions on temporary directory %s: %s", testDir, err)
	}
	// Ensure that os.RemoveAll does fail with perm error (i.e. our test dir is sane)
	err = os.RemoveAll(testDir)
	if err == nil {
		t.Fatalf("os.RemoveAll unexpectedly succeeded removing test directory %s", testDir)
	}
	if !os.IsPermission(err) {
		t.Fatalf("os.RemoveAll unexpectedly errored trying removing test directory %s: %s", testDir, err)
	}

	// Our Test - Ensure ForceRemoveAll does *not* fail & the directory is gone
	err = ForceRemoveAll(testDir)
	if err != nil {
		t.Errorf("ForceRemoveAll unexpectedly errored trying to remove test directory %s: %s", testDir, err)
	}
	ok, err := PathExists(testDir)
	if err != nil {
		t.Errorf("Error checking success of ForceRemoveAll on %s: %s", testDir, err)
	}
	if ok {
		t.Errorf("ForceRemoveAll failed to remove %s", testDir)
	}
}

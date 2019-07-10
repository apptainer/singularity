// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package fs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

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

	os.Symlink("usr/bin", filepath.Join(tmpdir, "bin"))
	os.Symlink("usr/sbin", filepath.Join(tmpdir, "sbin"))

	MkdirAll(filepath.Join(tmpdir, "usr", "bin"), 0755)
	MkdirAll(filepath.Join(tmpdir, "usr", "sbin"), 0755)

	os.Symlink("/bin", filepath.Join(tmpdir, "bin", "bin"))
	os.Symlink("../sbin", filepath.Join(tmpdir, "bin", "sbin"))
	os.Symlink("../../sbin", filepath.Join(tmpdir, "sbin", "sbin2"))

	testPath := []struct {
		path string
		eval string
	}{
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
		{"/fake/test", "/fake/test"},
	}

	for _, p := range testPath {
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

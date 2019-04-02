// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package instance

import (
	"os"
	"path/filepath"
	"testing"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/test"
)

const testSubDir = "testing"

func TestProcName(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	test := ProcName("test", "test")
	match := "Singularity instance: test [test]"
	if test != match {
		t.Errorf("unexpected match %s != %s", test, match)
	}
}

func TestExtractName(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		input  string
		output string
	}{
		{
			input:  "instance://test",
			output: "test",
		},
		{
			input:  "instance:/test",
			output: "instance:/test",
		},
		{
			input:  "instance:///test",
			output: "/test",
		},
	}
	for _, e := range tests {
		o := ExtractName(e.input)
		if o != e.output {
			t.Errorf("unexpected match, got %s instead of %s", o, e.output)
		}
	}
}

func TestCheckName(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		desc          string
		name          string
		expectFailure bool
	}{
		{
			desc:          "with valid name",
			name:          "test",
			expectFailure: false,
		},
		{
			desc:          "with valid name containg number",
			name:          "test123",
			expectFailure: false,
		},
		{
			desc:          "with invalid name containing space",
			name:          "test 123",
			expectFailure: true,
		},
		{
			desc:          "with valid name containing underscore",
			name:          "test_123",
			expectFailure: false,
		},
		{
			desc:          "with valid name containing dot",
			name:          "test.123",
			expectFailure: false,
		},
		{
			desc:          "with valid name containing minus",
			name:          "test-123",
			expectFailure: false,
		},
		{
			desc:          "with invalid name containing slash",
			name:          "test/123",
			expectFailure: true,
		},
	}
	for _, e := range tests {
		err := CheckName(e.name)
		if err != nil && !e.expectFailure {
			t.Errorf("unexpected failure %s: %s", e.desc, err)
		}
	}
}

func TestGetDirUnprivileged(t *testing.T) {
	test.EnsurePrivilege(t)

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("unable to retrieve hostname: %s", err)
	}

	instancePath := filepath.Join("/root", unprivPath, testSubDir, hostname, "root")

	tests := []struct {
		name          string
		path          string
		expectFailure bool
	}{
		{
			name:          "test",
			path:          filepath.Join(instancePath, "test"),
			expectFailure: false,
		},
		{
			name:          "test/123",
			expectFailure: true,
		},
	}
	for _, e := range tests {
		path, err := GetDirUnprivileged(e.name, testSubDir)
		if err != nil && !e.expectFailure {
			t.Errorf("unexpected failure for name %s: %s", e.name, err)
		} else if !e.expectFailure && path != e.path {
			t.Errorf("unexpected path returned %s instead of %s", path, e.path)
		}
	}
}

func TestGetDirPrivileged(t *testing.T) {
	test.EnsurePrivilege(t)

	instancePath := filepath.Join(privPath, testSubDir, "root")

	tests := []struct {
		name          string
		path          string
		expectFailure bool
	}{
		{
			name:          "test",
			path:          filepath.Join(instancePath, "test"),
			expectFailure: false,
		},
		{
			name:          "test/123",
			expectFailure: true,
		},
	}
	for _, e := range tests {
		path, err := GetDirPrivileged(e.name, testSubDir)
		if err != nil && !e.expectFailure {
			t.Errorf("unexpected failure for name %s: %s", e.name, err)
		} else if !e.expectFailure && path != e.path {
			t.Errorf("unexpected path returned %s instead of %s", path, e.path)
		}
	}
}

var instanceTests = []struct {
	name          string
	privileged    bool
	expectFailure bool
}{
	{
		name:          "valid_privileged_instance",
		privileged:    true,
		expectFailure: false,
	},
	{
		name:          "valid_privileged_instance",
		privileged:    true,
		expectFailure: true,
	},
	{
		name:          "valid_unprivileged_instance",
		privileged:    false,
		expectFailure: false,
	},
	{
		name:          "invalid_privileged_instance",
		privileged:    true,
		expectFailure: true,
	},
	{
		name:          "invalid_unprivileged_instance",
		privileged:    false,
		expectFailure: true,
	},
}

func TestAdd(t *testing.T) {
	test.EnsurePrivilege(t)

	for _, e := range instanceTests {
		var err error
		var file *File

		file, err = Add(e.name, e.privileged, testSubDir)
		if err != nil && !e.expectFailure {
			t.Errorf("unexpected failure for name %s: %s", e.name, err)
		}
		if file != nil {
			file.User = "root"
			file.Pid = os.Getpid()
			if err := file.Update(); err != nil {
				t.Errorf("error while creating instance %s: %s", e.name, err)
			}
			if err := file.MountNamespaces(); err != nil {
				t.Errorf("error while mounting namespaces: %s", err)
			}
			err := file.UpdateNamespacesPath([]specs.LinuxNamespace{})
			if err == nil {
				t.Errorf("unexpected success while updating namespace paths")
			}
			// should always fail with 'no command line match found'
			file.PPid = file.Pid
			err = file.UpdateNamespacesPath([]specs.LinuxNamespace{})
			if err == nil {
				t.Errorf("unexpected success while updating namespace paths")
			}
			stdout, stderr, err := SetLogFile(e.name, 0, testSubDir)
			if err != nil {
				t.Errorf("error while creating instance log file: %s", err)
			}
			if err := os.Remove(stdout.Name()); err != nil {
				t.Errorf("error while delete instance log out file: %s", err)
			}
			if err := os.Remove(stderr.Name()); err != nil {
				t.Errorf("error while deleting instance log err file: %s", err)
			}
		}
	}
}

func TestGet(t *testing.T) {
	test.EnsurePrivilege(t)

	for _, e := range instanceTests {
		var err error
		var file *File

		file, err = Get(e.name, testSubDir)
		if err != nil && !e.expectFailure {
			t.Errorf("unexpected failure for name %s: %s", e.name, err)
		}
		if file != nil {
			if file.User != "root" {
				t.Errorf("unexpected user returned %s", file.User)
			}
			if e.privileged && !file.PrivilegedPath() {
				t.Errorf("unexpected path for privileged instance")
			} else if !e.privileged && file.PrivilegedPath() {
				t.Errorf("unexpected path for unprivileged instance")
			}
			err = file.Delete()
			if err != nil && !e.expectFailure {
				t.Errorf("unexpected error while deleting instance %s: %s", e.name, err)
			}
		}
	}
}

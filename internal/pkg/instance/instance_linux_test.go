// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package instance

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

const testSubDir = "testing"

var fakeInstancePid int

func TestProcName(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		desc            string
		name            string
		user            string
		match           string
		expectedFailure bool
	}{
		{
			desc:  "with valid same name/username",
			name:  "test",
			user:  "test",
			match: "Singularity instance: test [test]",
		},
		{
			desc:  "with valid different name/username",
			name:  "instance",
			user:  "user",
			match: "Singularity instance: user [instance]",
		},
		{
			desc:            "with empty name",
			name:            "",
			user:            "test",
			expectedFailure: true,
		},
		{
			desc:            "with empty username",
			name:            "test",
			user:            "",
			expectedFailure: true,
		},
		{
			desc:            "both empty name/username",
			name:            "",
			user:            "",
			expectedFailure: true,
		},
	}
	for _, e := range tests {
		m, err := ProcName(e.name, e.user)
		if err != nil && !e.expectedFailure {
			t.Errorf("unexpected failure for test '%s': %s", e.desc, err)
		} else if err == nil && e.expectedFailure {
			t.Errorf("unexpected success for test '%s'", e.desc)
		} else if m != e.match {
			t.Errorf("unexpected match %s != %s", m, e.match)
		}
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
			t.Errorf("unexpected result, got %s instead of %s", o, e.output)
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
			desc:          "with valid name containing number",
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
		{
			desc:          "with empty name",
			name:          "",
			expectFailure: true,
		},
	}
	for _, e := range tests {
		err := CheckName(e.name)
		if err != nil && !e.expectFailure {
			t.Errorf("unexpected failure %s: %s", e.desc, err)
		} else if err == nil && e.expectFailure {
			t.Errorf("unexpected success %s", e.desc)
		}
	}
}

var instanceTests = []struct {
	name          string
	expectFailure bool
}{
	{
		name:          "valid_instance",
		expectFailure: false,
	},
	{
		name:          "valid_instance",
		expectFailure: true,
	},
	{
		name:          "invalid instance",
		expectFailure: true,
	},
}

func TestAdd(t *testing.T) {
	test.EnsurePrivilege(t)

	for _, e := range instanceTests {
		var err error
		var file *File

		file, err = Add(e.name, testSubDir)
		if err != nil && !e.expectFailure {
			t.Errorf("unexpected failure for name %s: %s", e.name, err)
		} else if err == nil && e.expectFailure {
			t.Errorf("unexpected success for name %s", e.name)
		}
		if file == nil {
			continue
		}
		file.User = "root"
		file.PPid = fakeInstancePid
		file.Pid = os.Getpid()
		if err := file.Update(); err != nil {
			t.Errorf("error while creating instance %s: %s", e.name, err)
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

func TestGet(t *testing.T) {
	test.EnsurePrivilege(t)

	for _, e := range instanceTests {
		var err error
		var file *File

		file, err = Get(e.name, testSubDir)
		if err != nil && !e.expectFailure {
			t.Errorf("unexpected failure for name %s: %s", e.name, err)
		} else if err == nil && e.expectFailure {
			t.Errorf("unexpected success for name %s", e.name)
		}
		if file == nil {
			continue
		}
		if file.User != "root" {
			t.Errorf("unexpected user returned %s", file.User)
		}
		path, err := GetDir(e.name, testSubDir)
		if err != nil {
			t.Errorf("unexpected error while retrieving instance directory path: %s", err)
		}
		instanceDir := filepath.Dir(file.Path)
		if path != instanceDir {
			t.Errorf("unexpected instance directory path, got %s instead of %s", path, instanceDir)
		}
		if file.isExited() {
			t.Errorf("fake instance is not running")
		}
		err = file.Delete()
		if err != nil && !e.expectFailure {
			t.Errorf("unexpected error while deleting instance %s: %s", e.name, err)
		}
	}
}

func TestMain(m *testing.M) {
	// spawn a fake instance process
	cmd := exec.Command("cat")
	// keep cat running until it gets killed
	cmd.StdinPipe()
	// set process to "Singularity instance"
	cmd.Args = []string{ProgPrefix}
	if err := cmd.Start(); err != nil {
		os.Exit(1)
	}
	fakeInstancePid = cmd.Process.Pid

	// execute tests
	e := m.Run()

	// kill the fake instance process
	cmd.Process.Kill()

	os.Exit(e)
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build seccomp

package seccomp

import (
	"io/ioutil"
	"os"
	"syscall"
	"testing"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sylabs/singularity/internal/pkg/test"
)

func defaultProfile() *specs.LinuxSeccomp {
	syscalls := []specs.LinuxSyscall{
		{
			Names:  []string{"fchmod"},
			Action: specs.ActErrno,
			Args: []specs.LinuxSeccompArg{
				{
					Index: 1,
					Value: 0777,
					Op:    specs.OpEqualTo,
				},
			},
		},
	}
	return &specs.LinuxSeccomp{
		DefaultAction: specs.ActAllow,
		Syscalls:      syscalls,
	}
}

func testFchmod(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "chmod_file")
	if err != nil {
		t.Fatal(err)
	}
	file := tmpfile.Name()

	defer os.Remove(file)
	defer tmpfile.Close()

	if hasConditionSupport() {
		if err := syscall.Fchmod(int(tmpfile.Fd()), 0755); err != nil {
			t.Errorf("fchmod syscall failed: %s", err)
		}
		if err := syscall.Fchmod(int(tmpfile.Fd()), 0777); err == nil {
			t.Errorf("fchmod syscall didn't return operation not permitted")
		}
	}
}

func TestLoadSeccompConfig(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	if err := LoadSeccompConfig(nil, false); err == nil {
		t.Errorf("shoud have failed with an empty config")
	}
	if err := LoadSeccompConfig(defaultProfile(), true); err != nil {
		t.Errorf("%s", err)
	}

	testFchmod(t)
}

func TestLoadProfileFromFile(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	gen := &generate.Generator{Config: &specs.Spec{}}

	if err := LoadProfileFromFile("test_profile/fake.json", gen); err == nil {
		t.Errorf("shoud have failed with inexistent file")
	}

	if err := LoadProfileFromFile("test_profile/test.json", gen); err != nil {
		t.Error(err)
	}

	testFchmod(t)
}

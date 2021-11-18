// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

//go:build seccomp
// +build seccomp

package seccomp

import (
	"errors"
	"io/ioutil"
	"os"
	"reflect"
	"syscall"
	"testing"

	"github.com/hpcng/singularity/internal/pkg/runtime/engine/config/oci/generate"
	"github.com/hpcng/singularity/internal/pkg/test"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	lseccomp "github.com/seccomp/libseccomp-golang"
)

func defaultProfile() *specs.LinuxSeccomp {
	syscalls := []specs.LinuxSyscall{
		{
			Names:  []string{"fchmod"},
			Action: specs.ActErrno,
			Args: []specs.LinuxSeccompArg{
				{
					Index: 1,
					Value: 0o777,
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
		// all modes except 0777 are permitted
		if err := syscall.Fchmod(int(tmpfile.Fd()), 0o755); err != nil {
			t.Errorf("fchmod syscall failed: %s", err)
		}
		if err := syscall.Fchmod(int(tmpfile.Fd()), 0o777); err == nil {
			t.Errorf("fchmod syscall didn't return operation not permitted")
		}
	} else {
		if err := syscall.Fchmod(int(tmpfile.Fd()), 0o755); err == nil {
			t.Errorf("fchmod syscall didn't return operation not permitted")
		}
	}
}

func TestLoadSeccompConfig(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	if err := LoadSeccompConfig(nil, false); err == nil {
		t.Errorf("should have failed with an empty config")
	}
	if err := LoadSeccompConfig(defaultProfile(), true); err != nil {
		t.Errorf("%s", err)
	}

	testFchmod(t)
}

func TestLoadProfileFromFile(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	gen := generate.New(nil)

	if err := LoadProfileFromFile("test_profile/fake.json", gen); err == nil {
		t.Errorf("should have failed with inexistent file")
	}

	if err := LoadProfileFromFile("test_profile/test.json", gen); err != nil {
		t.Error(err)
	}

	if err := LoadSeccompConfig(gen.Config.Linux.Seccomp, true); err != nil {
		t.Errorf("%s", err)
	}

	testFchmod(t)
}

func TestGetDefaultErrno(t *testing.T) {
	eperm := uint(syscall.EPERM)
	enosys := uint(syscall.ENOSYS)

	tests := []struct {
		name        string
		specs       *specs.LinuxSeccomp
		expectErrno *uint
		expectError error
	}{
		{
			name:        "EmptyDefaultEPERM",
			specs:       &specs.LinuxSeccomp{},
			expectErrno: &eperm,
			expectError: nil,
		},
		{
			name: "ActErrnoDefaultEPERM",
			specs: &specs.LinuxSeccomp{
				DefaultAction: specs.ActErrno,
			},
			expectErrno: &eperm,
			expectError: nil,
		},
		{
			name: "ActTraceDefaultEPERM",
			specs: &specs.LinuxSeccomp{
				DefaultAction: specs.ActTrace,
			},
			expectErrno: &eperm,
			expectError: nil,
		},
		{
			name: "ActKillDefaultEPERM",
			specs: &specs.LinuxSeccomp{
				DefaultAction: specs.ActKill,
			},
			expectErrno: &eperm,
			expectError: nil,
		},
		{
			name: "ActErrnoENOSYS",
			specs: &specs.LinuxSeccomp{
				DefaultAction:   specs.ActErrno,
				DefaultErrnoRet: &enosys,
			},
			expectErrno: &enosys,
			expectError: nil,
		},
		{
			name: "ActTraceENOSYS",
			specs: &specs.LinuxSeccomp{
				DefaultAction:   specs.ActTrace,
				DefaultErrnoRet: &enosys,
			},
			expectErrno: &enosys,
			expectError: nil,
		},
		{
			name: "ActKillENOSYS",
			specs: &specs.LinuxSeccomp{
				DefaultAction:   specs.ActKill,
				DefaultErrnoRet: &enosys,
			},
			expectErrno: nil,
			expectError: ErrUnsupportedErrno,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errno, err := getDefaultErrno(tt.specs)

			if err == nil {
				if tt.expectError != nil {
					t.Errorf("no error, expected %v", tt.expectError)
				}
			} else {
				if !errors.Is(err, tt.expectError) {
					t.Errorf("got err=%v, expected %v", err, tt.expectError)
				}
			}

			if !reflect.DeepEqual(errno, tt.expectErrno) {
				t.Errorf("got errno=%v, expected %v", errno, tt.expectErrno)
			}
		})
	}
}

func TestGetAction(t *testing.T) {
	eperm := uint(syscall.EPERM)
	enosys := uint(syscall.ENOSYS)

	errnoEPERM := lseccomp.ActErrno
	errnoEPERM = errnoEPERM.SetReturnCode(int16(syscall.EPERM))
	errnoENOSYS := lseccomp.ActErrno
	errnoENOSYS = errnoENOSYS.SetReturnCode(int16(syscall.ENOSYS))
	traceEPERM := lseccomp.ActTrace
	traceEPERM = traceEPERM.SetReturnCode(int16(syscall.EPERM))
	traceENOSYS := lseccomp.ActTrace
	traceENOSYS = traceENOSYS.SetReturnCode(int16(syscall.ENOSYS))
	kill := lseccomp.ActKill

	tests := []struct {
		name         string
		action       specs.LinuxSeccompAction
		errno        *uint
		defaultErrno uint
		expectAction lseccomp.ScmpAction
		expectError  error
	}{
		{
			name:         "KillOK",
			action:       specs.ActKill,
			errno:        nil,
			defaultErrno: eperm,
			expectAction: kill,
			expectError:  nil,
		},
		{
			name:         "KillUnsupportedErrno",
			action:       specs.ActKill,
			errno:        &enosys,
			defaultErrno: eperm,
			expectAction: lseccomp.ActInvalid,
			expectError:  ErrUnsupportedErrno,
		},
		{
			name:         "ErrnoDefaultEPERM",
			action:       specs.ActErrno,
			errno:        nil,
			defaultErrno: eperm,
			expectAction: errnoEPERM,
			expectError:  nil,
		},
		{
			name:         "ErrnoOverrideENOSYS",
			action:       specs.ActErrno,
			errno:        &enosys,
			defaultErrno: eperm,
			expectAction: errnoENOSYS,
			expectError:  nil,
		},
		{
			name:         "TraceDefaultEPERM",
			action:       specs.ActTrace,
			errno:        nil,
			defaultErrno: eperm,
			expectAction: traceEPERM,
			expectError:  nil,
		},
		{
			name:         "TraceOverrideENOSYS",
			action:       specs.ActTrace,
			errno:        &enosys,
			defaultErrno: eperm,
			expectAction: traceENOSYS,
			expectError:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := getAction(tt.action, tt.errno, tt.defaultErrno)

			if err == nil {
				if tt.expectError != nil {
					t.Errorf("no error, expected %v", tt.expectError)
				}
			} else {
				if !errors.Is(err, tt.expectError) {
					t.Errorf("got err=%v, expected %v", err, tt.expectError)
				}
			}

			if action != tt.expectAction {
				t.Errorf("got action=%v, expected %v", action, tt.expectAction)
			}
		})
	}
}

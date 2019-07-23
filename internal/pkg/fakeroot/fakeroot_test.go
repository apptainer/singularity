// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package fakeroot

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/user"
)

type set struct {
	name            string
	path            string
	uid             uint32
	expectedMapping *specs.LinuxIDMapping
}

func testGetIDRange(t *testing.T, s set) {
	idRange, err := GetIDRange(s.path, s.uid)
	if err != nil && s.expectedMapping != nil {
		t.Errorf("unexpected error for %q: %s", s.name, err)
	} else if err == nil && s.expectedMapping == nil {
		t.Errorf("unexpected success for %q", s.name)
	} else if err == nil && s.expectedMapping != nil {
		if s.expectedMapping.ContainerID != idRange.ContainerID {
			t.Errorf("bad container ID returned for %q: %d instead of %d", s.name, idRange.ContainerID, s.expectedMapping.ContainerID)
		}
		if s.expectedMapping.HostID != idRange.HostID {
			t.Errorf("bad host ID returned for %q: %d instead of %d", s.name, idRange.HostID, s.expectedMapping.HostID)
		}
		if s.expectedMapping.Size != idRange.Size {
			t.Errorf("bad size returned for %q: %d instead of %d", s.name, idRange.Size, s.expectedMapping.Size)
		}
	}
}

func TestGetIDRangePath(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	f, err := fs.MakeTmpFile("", "subid-", 0700)
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(f.Name())

	var subIDContent = `
root:100000:65536
1:165536:1
1:165536:65536
2:2000000:-1
3:-1:65536
4:2065536:1
`

	f.WriteString(subIDContent)
	f.Close()

	tests := []set{
		{
			name: "empty path",
			path: "",
			uid:  0,
		},
		{
			name: "bad path",
			path: "/a/bad/path",
			uid:  0,
		},
		{
			name: "temporary file, bad uid",
			path: f.Name(),
			uid:  ^uint32(0),
		},
		{
			name: "temporary file, user root (good)",
			path: f.Name(),
			uid:  0,
			expectedMapping: &specs.LinuxIDMapping{
				ContainerID: 1,
				HostID:      100000,
				Size:        65536,
			},
		},
		{
			name: "temporary file, uid 1 (multiple good)",
			path: f.Name(),
			uid:  1,
			expectedMapping: &specs.LinuxIDMapping{
				ContainerID: 1,
				HostID:      165536,
				Size:        65536,
			},
		},
		{
			name: "temporary file, uid 2 (bad size)",
			path: f.Name(),
			uid:  2,
		},
		{
			name: "temporary file, uid 2 (bad containerID)",
			path: f.Name(),
			uid:  3,
		},
		{
			name: "temporary file, uid 4 (multiple bad)",
			path: f.Name(),
			uid:  4,
		},
		{
			name: "temporary file, uid 8 (doesn't exist)",
			path: f.Name(),
			uid:  8,
		},
	}
	for _, test := range tests {
		testGetIDRange(t, test)
	}
}

func getUserFn(username string) (*user.User, error) {
	var prefix string

	splitted := strings.Split(username, "_")
	prefix = splitted[0]
	uid, err := strconv.Atoi(splitted[1])
	if err != nil {
		return nil, err
	}
	if prefix == "nouser" {
		return nil, fmt.Errorf("%s not found", username)
	}

	return &user.User{
		Name: prefix,
		UID:  uint32(uid),
	}, nil
}

func createConfig(t *testing.T) string {
	f, err := fs.MakeTmpFile("", "subid-", 0644)
	if err != nil {
		t.Fatalf("failed to create temporary config: %s", err)
	}
	defer f.Close()

	var buf bytes.Buffer

	base := 10
	size := 10

	// valid users
	for i := base; i < base+size; i++ {
		line := fmt.Sprintf("valid_%d:%d:%d\n", i, startMax-((i-base)*validRangeCount), validRangeCount)
		buf.WriteString(line)
	}
	buf.WriteString("\n")
	// badstart users
	base += size
	for i := base; i < base+size; i++ {
		line := fmt.Sprintf("badstart_%d:%d:%d\n", i, -1, validRangeCount)
		buf.WriteString(line)
	}
	buf.WriteString("\n")
	// badcount users
	base += size
	for i := base; i < base+size; i++ {
		line := fmt.Sprintf("badcount_%d:%d:%d\n", i, (i+1)*validRangeCount, 0)
		buf.WriteString(line)
	}
	buf.WriteString("\n")
	// disabled users
	base += size
	for i := base; i < base+size; i++ {
		line := fmt.Sprintf("!disabled_%d:%d:%d\n", i, (i+1)*validRangeCount, validRangeCount)
		buf.WriteString(line)
	}
	// same user
	base += size
	for i := base; i < base+size; i++ {
		line := fmt.Sprintf("sameuser_%d:%d:%d\n", base, (i+1)*validRangeCount, 1)
		buf.WriteString(line)
	}
	// add a bad formatted entry
	buf.WriteString("badentry:\n")
	// add a nouser entry
	buf.WriteString("nouser_42:0:0\n")

	if _, err := f.Write(buf.Bytes()); err != nil {
		t.Fatalf("failed to write config: %s", err)
	}

	return f.Name()
}

func testGetUserEntry(t *testing.T, config *Config) {
	tests := []struct {
		desc           string
		username       string
		reportBadEntry bool
		expectSuccess  bool
	}{
		{
			desc:           "ValidUser",
			username:       "valid_10",
			reportBadEntry: false,
			expectSuccess:  true,
		},
		{
			desc:           "ValidUserReportBadEntry",
			username:       "valid_10",
			reportBadEntry: true,
			expectSuccess:  true,
		},
		{
			desc:           "NoUser",
			username:       "nouser_10",
			reportBadEntry: false,
			expectSuccess:  false,
		},
		{
			desc:           "NoUserReportBadEntry",
			username:       "nouser_10",
			reportBadEntry: true,
			expectSuccess:  false,
		},
		{
			desc:           "BadStartUser",
			username:       "badstart_20",
			reportBadEntry: false,
			expectSuccess:  false,
		},
		{
			desc:           "BadStartUserReportBadEntry",
			username:       "badstart_20",
			reportBadEntry: true,
			expectSuccess:  false,
		},
		{
			desc:           "DisabledUser",
			username:       "disabled_40",
			reportBadEntry: false,
			expectSuccess:  true,
		},
		{
			desc:           "DisabledUserReportBadEntry",
			username:       "disabled_40",
			reportBadEntry: true,
			expectSuccess:  true,
		},
		{
			desc:           "SameUser",
			username:       "sameuser_50",
			reportBadEntry: false,
			expectSuccess:  false,
		},
		{
			desc:           "SameUserReportBadEntry",
			username:       "sameuser_50",
			reportBadEntry: true,
			expectSuccess:  false,
		},
	}
	for _, tt := range tests {
		_, err := config.GetUserEntry(tt.username, tt.reportBadEntry)
		if err != nil && tt.expectSuccess {
			t.Errorf("unexpected error for %q: %s", tt.desc, err)
		} else if err == nil && !tt.expectSuccess {
			t.Errorf("unexpected success for %q", tt.desc)
		}
	}
}

func testEditEntry(t *testing.T, config *Config) {
	tests := []struct {
		desc          string
		username      string
		editFn        func(string) error
		expectSuccess bool
	}{
		{
			desc:          "AddNoUser",
			username:      "nouser_10",
			editFn:        config.AddUser,
			expectSuccess: false,
		},
		{
			desc:          "RemoveNoUser",
			username:      "nouser_10",
			editFn:        config.RemoveUser,
			expectSuccess: false,
		},
		{
			desc:          "EnableNoUser",
			username:      "nouser_10",
			editFn:        config.EnableUser,
			expectSuccess: false,
		},
		{
			desc:          "DisableNoUser",
			username:      "nouser_10",
			editFn:        config.DisableUser,
			expectSuccess: false,
		},
		{
			desc:          "AddAnotherValidUser",
			username:      "valid_100",
			editFn:        config.AddUser,
			expectSuccess: true,
		},
		{
			desc:          "RemoveAnotherValidUser",
			username:      "valid_100",
			editFn:        config.RemoveUser,
			expectSuccess: true,
		},
		{
			desc:          "AddSameValidUser",
			username:      "valid_10",
			editFn:        config.AddUser,
			expectSuccess: true,
		},
		{
			desc:          "DisableValidUser",
			username:      "valid_11",
			editFn:        config.DisableUser,
			expectSuccess: true,
		},
		{
			desc:          "DisableSameValidUser",
			username:      "valid_11",
			editFn:        config.DisableUser,
			expectSuccess: true,
		},
		{
			desc:          "EnableDisabledUser",
			username:      "disabled_40",
			editFn:        config.EnableUser,
			expectSuccess: true,
		},
		{
			desc:          "EnableSameDisabledValidUser",
			username:      "disabled_40",
			editFn:        config.EnableUser,
			expectSuccess: true,
		},
		{
			desc:          "RemoveValidUser",
			username:      "valid_10",
			editFn:        config.RemoveUser,
			expectSuccess: true,
		},
		{
			desc:          "RemoveSameValidUser",
			username:      "valid_10",
			editFn:        config.RemoveUser,
			expectSuccess: false,
		},
		{
			desc:          "AddAnotherValidUser",
			username:      "valid_21",
			editFn:        config.AddUser,
			expectSuccess: true,
		},
	}
	for _, tt := range tests {
		err := tt.editFn(tt.username)
		if err != nil && tt.expectSuccess {
			t.Errorf("unexpected error for %q: %s", tt.desc, err)
		} else if err == nil && !tt.expectSuccess {
			t.Errorf("unexpected success for %q", tt.desc)
		}
	}

	file := config.file.Name()

	config.Close()

	// basic checks to verify that write works correctly
	config, err := GetConfig(file, true, getUserFn)
	if err != nil {
		t.Fatalf("unexpected error while getting config %s: %s", file, err)
	}
	defer config.Close()

	// this entry was removed
	if _, err := config.GetUserEntry("valid_10", false); err == nil {
		t.Errorf("unexpected entry found for valid_10 user")
	}
	// this entry was disabled
	e, err := config.GetUserEntry("valid_11", false)
	if err != nil {
		t.Errorf("unexpected error for valid_11 user")
	}
	if !e.disabled {
		t.Errorf("valid_11 user entry should be disabled")
	}
	// this entry was enabled
	e, err = config.GetUserEntry("disabled_40", false)
	if err != nil {
		t.Errorf("unexpected error for disabled_40 user")
	}
	if e.disabled {
		t.Errorf("disabled_40 user entry should be enabled")
	}
	// this entry was added and range start should be
	// equal to startMax (as it replace valid_10)
	e, err = config.GetUserEntry("valid_21", false)
	if err != nil {
		t.Errorf("unexpected error for valid_21 user")
	}
	if e.Start != startMax {
		t.Errorf("valid_21 user entry start range should be %d, got %d", startMax, e.Start)
	}
}

func TestConfig(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	file := createConfig(t)
	defer os.Remove(file)

	// test with empty path
	_, err := GetConfig("", true, nil)
	if err == nil {
		t.Fatalf("unexpected success while getting empty: %s", err)
	}

	config, err := GetConfig(file, true, getUserFn)
	if err != nil {
		t.Fatalf("unexpected error while getting config %s: %s", file, err)
	}

	testGetUserEntry(t, config)
	testEditEntry(t, config)
}

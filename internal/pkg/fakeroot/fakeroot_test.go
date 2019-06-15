// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package fakeroot

import (
	"os"
	"testing"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/internal/pkg/util/user"
)

type set struct {
	name            string
	base            uint64
	allowedUsers    []string
	expectedMapping *specs.LinuxIDMapping
}

func testGetIDRange(t *testing.T, s set) {
	idRange, err := GetIDRange(s.base, s.allowedUsers)
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

func TestGetIDRangeUser(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	userinfo, err := user.GetPwUID(uint32(os.Getuid()))
	if err != nil {
		t.Fatalf("failed to retrieve user information: %s", err)
	}

	tests := []set{
		{
			name: "empty",
		},
		{
			name: "low base",
			base: 65535,
		},
		{
			name: "high base",
			base: 65536 * 65536,
		},
		{
			name: "base not multiple of 65536",
			base: 65537,
		},
		{
			name: "good base, no users",
			base: 65536,
		},
		{
			name:         "good base, current user",
			base:         65536 * 1024,
			allowedUsers: []string{userinfo.Name},
			expectedMapping: &specs.LinuxIDMapping{
				ContainerID: 1,
				HostID:      65536 * 1024,
				Size:        65535,
			},
		},
	}
	for _, test := range tests {
		testGetIDRange(t, test)
	}
}

func TestGetIDRangeRoot(t *testing.T) {
	test.EnsurePrivilege(t)

	tests := []set{
		{
			name: "empty",
		},
		{
			name: "low base",
			base: 65535,
		},
		{
			name: "high base",
			base: 65536 * 65536,
		},
		{
			name: "base not multiple of 65536",
			base: 65537,
		},
		{
			name: "good base, root user",
			base: 65536 * 1024,
			expectedMapping: &specs.LinuxIDMapping{
				ContainerID: 1,
				HostID:      1,
				Size:        65535,
			},
		},
	}
	for _, test := range tests {
		testGetIDRange(t, test)
	}
}

// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package fakeroot

import (
	"io/ioutil"
	"os"
	"testing"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/test"
)

type set struct {
	name            string
	path            string
	uid             uint32
	expectedMapping *specs.LinuxIDMapping
}

var subIDContent = `
root:100000:65536
1:165536:1
1:165536:65536
2:2000000:-1
3:-1:65536
4:2065536:1
`

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

	f, err := ioutil.TempFile("", "subid-")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(f.Name())

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

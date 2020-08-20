// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package capabilities

import (
	"runtime"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestGetProcess(t *testing.T) {
	test.EnsurePrivilege(t)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	tests := []struct {
		name string
		fn   func() (uint64, error)
		cap  string
	}{
		{
			name: "effective",
			fn:   GetProcessEffective,
			cap:  "CAP_SYS_ADMIN",
		},
		{
			name: "permitted",
			fn:   GetProcessPermitted,
		},
		{
			name: "inheritable",
			fn:   GetProcessInheritable,
		},
	}

	for _, tt := range tests {
		caps, err := tt.fn()
		if err != nil {
			t.Fatalf("unexpected error while getting process %s capabilities: %s", tt.name, err)
		}
		cap := Map[tt.cap]
		if tt.cap != "" && caps&uint64(1<<cap.Value) == 0 {
			t.Fatalf("%s capability %s missing", tt.name, tt.cap)
		}
	}
}

func TestSetProcessEffective(t *testing.T) {
	test.EnsurePrivilege(t)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	data, err := getProcessCapabilities()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		cap          uint64
		oldEffective uint64
	}{
		{
			name:         "set cap_sys_admin only",
			cap:          uint64(1 << Map["CAP_SYS_ADMIN"].Value),
			oldEffective: uint64(data[0].Effective) | uint64(data[1].Effective)<<32,
		},
		{
			name:         "restore capabilities",
			cap:          uint64(data[0].Effective) | uint64(data[1].Effective)<<32,
			oldEffective: uint64(1 << Map["CAP_SYS_ADMIN"].Value),
		},
	}

	for _, tt := range tests {
		old, err := SetProcessEffective(tt.cap)
		if err != nil {
			t.Fatalf("unexpected error for %s: %s", tt.name, err)
		} else if old != tt.oldEffective {
			t.Fatalf("unexpected old effective set for %s", tt.name)
		}
	}
}

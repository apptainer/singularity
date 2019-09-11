// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package user

import (
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestGetPwUID(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	u, err := GetPwUID(0)
	if err != nil {
		t.Fatalf("Failed to retrieve information for UID 0")
	}
	if u.Name != "root" {
		t.Fatalf("UID 0 doesn't correspond to root user")
	}
}

func TestGetPwNam(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	u, err := GetPwNam("root")
	if err != nil {
		t.Fatalf("Failed to retrieve information for root user")
	}
	if u.UID != 0 {
		t.Fatalf("root user doesn't have UID 0")
	}
}

func TestGetGrGID(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	group, err := GetGrGID(0)
	if err != nil {
		t.Fatalf("Failed to retrieve information for GID 0")
	}
	if group.Name != "root" {
		t.Fatalf("GID 0 doesn't correspond to root group")
	}
}

func TestGetGrNam(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	group, err := GetGrNam("root")
	if err != nil {
		t.Fatalf("Failed to retrieve information for root group")
	}
	if group.GID != 0 {
		t.Fatalf("root group doesn't have GID 0")
	}
}

func testCurrent(t *testing.T, fn func() (*User, error)) {
	uid := os.Getuid()

	u, err := fn()
	if err != nil {
		t.Fatalf("Failed to retrieve information for current user")
	}
	if u.UID != uint32(uid) {
		t.Fatalf("returned UID (%d) doesn't match current UID (%d)", uid, u.UID)
	}
}

func TestCurrent(t *testing.T) {
	// as a regular user
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	testCurrent(t, Current)

	// as root
	test.ResetPrivilege(t)

	testCurrent(t, Current)
}

func TestCurrentOriginal(t *testing.T) {
	// as a regular user
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// to fully test CurrentOriginal, we would need
	// to execute it from a user namespace, actually
	// we just ensure that current user informations
	// are returned
	testCurrent(t, CurrentOriginal)

	// as root
	test.ResetPrivilege(t)

	testCurrent(t, CurrentOriginal)
}

/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE.md file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package user

import (
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestGetPwUID(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	user, err := GetPwUID(0)
	if err != nil {
		t.Fatalf("Failed to retrieve information for UID 0")
	}
	if user.Name != "root" {
		t.Fatalf("UID 0 doesn't correspond to root user")
	}
}

func TestGetPwNam(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	user, err := GetPwNam("root")
	if err != nil {
		t.Fatalf("Failed to retrieve information for root user")
	}
	if user.UID != 0 {
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

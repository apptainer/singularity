// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package capabilities

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestSplit(t *testing.T) {
	testCaps := []struct {
		caps   string
		length int
	}{
		{"chown, sys_admin", 2},
		{"CAP_,     sys_admin        ", 1},
		{"cap_sys_admin, cap_chown", 2},
		{"CAP_sys_admin,CHOWN", 2},
	}
	for _, tc := range testCaps {
		caps, _ := Split(tc.caps)
		if len(caps) != tc.length {
			t.Errorf("should have returned %d as capability len instead of %d", tc.length, len(caps))
		}
	}
}

func TestOpen(t *testing.T) {
	validCaps := []string{
		"CAP_CHOWN",
		"CAP_SYS_ADMIN",
		"CAP_DAC_OVERRIDE",
	}
	invalidCaps := []string{
		"CAP_",
	}

	// test with empty file
	file, err := Open("", false)
	if err == nil {
		t.Errorf("should have failed with no such file or directory")
	}

	// create temporary
	tmpfile, err := ioutil.TempFile("", "capabilities-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	file, err = Open(tmpfile.Name(), false)
	if err != nil {
		t.Fatal(err)
	}

	if err := file.AddUserCaps("test", validCaps); err != nil {
		t.Error(err)
	}
	if err := file.AddUserCaps("test", invalidCaps); err == nil {
		t.Errorf("should have failed with unknown capability")
	}

	if err := file.AddGroupCaps("test", validCaps); err != nil {
		t.Error(err)
	}
	if err := file.AddGroupCaps("test", invalidCaps); err == nil {
		t.Errorf("should have failed with unknown capability")
	}

	users, groups := file.ListAllCaps()
	if len(users) != 1 {
		t.Errorf("should have returnes 1 instead of %d", len(users))
	}
	if len(groups) != 1 {
		t.Errorf("should have returnes 1 instead of %d", len(groups))
	}
	if len(users["test"]) != len(validCaps) {
		t.Errorf("should have returnes %d instead of %d", len(users["test"]), len(validCaps))
	}
	if len(groups["test"]) != len(validCaps) {
		t.Errorf("should have returnes %d instead of %d", len(groups["test"]), len(validCaps))
	}

	authorized, unauthorized := file.CheckUserCaps("test", validCaps)
	if len(authorized) != len(validCaps) {
		t.Errorf("should have returned %d instead of %d", len(validCaps), len(authorized))
	}
	if len(unauthorized) != 0 {
		t.Errorf("should have returned 0 instead of %d", len(unauthorized))
	}
	authorized, unauthorized = file.CheckGroupCaps("test", validCaps)
	if len(authorized) != len(validCaps) {
		t.Errorf("should have returned %d instead of %d", len(validCaps), len(authorized))
	}
	if len(unauthorized) != 0 {
		t.Errorf("should have returned 0 instead of %d", len(unauthorized))
	}

	if err := file.Write(); err != nil {
		t.Error(err)
	}

	if err := file.DropUserCaps("test", invalidCaps); err == nil {
		t.Errorf("should have failed with unknown capability")
	}
	if err := file.DropUserCaps("test2", validCaps); err == nil {
		t.Errorf("should have failed with unknown user")
	}
	if err := file.DropUserCaps("test", validCaps); err != nil {
		t.Error(err)
	}

	if err := file.DropGroupCaps("test", invalidCaps); err == nil {
		t.Errorf("should have failed with unknown capability")
	}
	if err := file.DropGroupCaps("test2", validCaps); err == nil {
		t.Errorf("should have failed with unknown group")
	}
	if err := file.DropGroupCaps("test", validCaps); err != nil {
		t.Error(err)
	}

	if err := file.Write(); err != nil {
		t.Error(err)
	}

	if err := file.Close(); err != nil {
		t.Error(err)
	}

	file, err = Open(tmpfile.Name(), true)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Write(); err == nil {
		t.Fatalf("should have failed since file was open in read-only mode")
	}

	if len(file.ListUserCaps("test")) != 0 {
		t.Errorf("should have returned 0 instead of %d", len(file.ListUserCaps("test")))
	}
	if len(file.ListGroupCaps("test")) != 0 {
		t.Errorf("should have returned 0 instead of %d", len(file.ListGroupCaps("test")))
	}

	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

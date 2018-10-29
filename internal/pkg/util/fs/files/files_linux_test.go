// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package files

import (
	"bytes"
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestGroup(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	var gids []int
	uid := os.Getuid()

	_, err := Group("/fake", uid, gids)
	if err == nil {
		t.Errorf("should have failed with bad group file")
	}
	_, err = Group("/etc/group", uid, gids)
	if err != nil {
		t.Errorf("should have passed with correct group file")
	}
}

func TestPasswd(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	uid := os.Getuid()

	_, err := Passwd("/fake", "/fake", uid)
	if err == nil {
		t.Errorf("should have failed with bad passwd file")
	}
	_, err = Passwd("/etc/passwd", "/home", uid)
	if err != nil {
		t.Errorf("should have passed with correct passwd file")
	}
}

func TestHostname(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	content, err := Hostname("")
	if err == nil {
		t.Errorf("should have failed with empty hostname")
	}
	content, err = Hostname("mycontainer")
	if err != nil {
		t.Errorf("should have passed with correct hostname")
	}
	if bytes.Compare(content, []byte("mycontainer\n")) != 0 {
		t.Errorf("Hostname returns a bad content")
	}
	content, err = Hostname("bad|hostname")
	if err == nil {
		t.Errorf("should have failed with non valid hostname")
	}
}

func TestResolvConf(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	content, err := ResolvConf([]string{})
	if err == nil {
		t.Errorf("should have failed with empty dns")
	}
	content, err = ResolvConf([]string{"test"})
	if err == nil {
		t.Errorf("should have failed with bad dns")
	}
	content, err = ResolvConf([]string{"8.8.8.8"})
	if err != nil {
		t.Errorf("should have passed with valid dns")
	}
	if bytes.Compare(content, []byte("nameserver 8.8.8.8\n")) != 0 {
		t.Errorf("ResolvConf returns a bad content")
	}
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package layout

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

func TestLayout(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	uid := os.Getuid()
	gid := os.Getgid()

	session := &Manager{}

	groups, err := os.Getgroups()
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range groups {
		if g != gid {
			gid = g
			break
		}
	}

	dir, err := ioutil.TempDir("", "session")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := session.AddDir("/etc"); err == nil {
		t.Errorf("should have failed with uninitialized root path")
	}
	if err := session.AddFile("/etc/passwd", nil); err == nil {
		t.Errorf("should have failed with uninitialized root path")
	}
	if err := session.AddSymlink("/etc/symlink", "/etc/passwd"); err == nil {
		t.Errorf("should have failed with uninitialized root path")
	}
	if err := session.Create(); err == nil {
		t.Errorf("should have failed with uninitialized root path")
	}

	if err := session.SetRootPath("/fakedirectory"); err == nil {
		t.Error("shoud have failed with invalid root path directory")
	}
	if err := session.SetRootPath(dir); err != nil {
		t.Fatal(err)
	}
	if err := session.SetRootPath(dir); err == nil {
		t.Error("shoud have failed with root path already set error")
	}

	if err := session.AddDir("etc"); err == nil {
		t.Errorf("should have failed with non absolute path")
	}
	if err := session.AddDir("/etc"); err != nil {
		t.Error(err)
	}
	if err := session.AddDir("/etc"); err == nil {
		t.Error("shoud have failed with existent path")
	}

	if _, err := session.GetPath("/etcd"); err == nil {
		t.Errorf("should have failed with non existent path")
	}

	if err := session.AddFile("/etc/passwd", []byte("hello")); err != nil {
		t.Error(err)
	}
	if err := session.AddSymlink("/etc/symlink", "/etc/passwd"); err != nil {
		t.Error(err)
	}

	if err := session.Chmod("/etc", 0777); err != nil {
		t.Error(err)
	}
	if err := session.Chmod("/etcd", 0777); err == nil {
		t.Error("should have failed with non existent path")
	}

	if err := session.Chown("/etc", uid, gid); err != nil {
		t.Error(err)
	}
	if err := session.Chown("/etcd", uid, gid); err == nil {
		t.Error("should have failed with non existent path")
	}

	if err := session.Chmod("/etc/passwd", 0600); err != nil {
		t.Error(err)
	}
	if err := session.Chown("/etc/passwd", uid, gid); err != nil {
		t.Error(err)
	}
	if err := session.Chown("/etc/symlink", uid, gid); err != nil {
		t.Error(err)
	}

	if err := session.Create(); err != nil {
		t.Fatal(err)
	}
	if p, err := session.GetPath("/etc"); err == nil {
		if !fs.IsDir(p) {
			t.Errorf("failed to create directory %s", p)
		}
	} else {
		t.Error(err)
	}
	if p, err := session.GetPath("/etc/passwd"); err != nil {
		t.Error(err)
	} else {
		if !fs.IsFile(p) {
			t.Errorf("failed to create file %s", p)
		}
	}
	if p, err := session.GetPath("/etc/symlink"); err != nil {
		t.Error(err)
	} else {
		if !fs.IsLink(p) {
			t.Errorf("failed to create symlink %s", p)
		}
	}

	if err := session.AddSymlink("/etc/symlink2", "/etc/passwd"); err != nil {
		t.Error(err)
	}
	if err := session.Update(); err != nil {
		t.Fatal(err)
	}
	if p, err := session.GetPath("/etc/symlink2"); err != nil {
		t.Error(err)
	} else {
		if !fs.IsLink(p) {
			t.Errorf("failed to create symlink %s", p)
		}
	}
}

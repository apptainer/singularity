// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cgroups

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/hpcng/singularity/internal/pkg/test"
	"github.com/hpcng/singularity/internal/pkg/test/tool/require"
)

func TestCgroupsV1(t *testing.T) {
	test.EnsurePrivilege(t)
	require.CgroupsV1(t)

	cmd := exec.Command("/bin/cat", "/dev/zero")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	pid := cmd.Process.Pid
	strPid := strconv.Itoa(pid)
	path := filepath.Join("/singularity", strPid)

	manager := &ManagerV1{pid: pid, path: path}

	cgroupsToml := "example/cgroups.toml"
	// Some systems, e.g. ppc64le may not have a 2MB page size, so don't
	// apply a 2MB hugetlb limit if that's the case.
	_, err := os.Stat("/sys/fs/cgroup/hugetlb/hugetlb.2MB.limit_in_bytes")
	if os.IsNotExist(err) {
		t.Log("No hugetlb.2MB.limit_in_bytes - using alternate cgroups test file")
		cgroupsToml = "example/cgroups-no-hugetlb.toml"
	}

	if err := manager.ApplyFromFile(cgroupsToml); err != nil {
		t.Fatal(err)
	}
	defer manager.Remove()

	rootPath := manager.GetCgroupRootPath()
	if rootPath == "" {
		t.Fatalf("can't determine cgroups root path, is cgroups enabled ?")
	}

	cpuShares := filepath.Join(rootPath, "cpu", path, "cpu.shares")
	ensureIntInFile(t, cpuShares, 1024)

	content := []byte("[cpu]\nshares = 512")
	tmpfile, err := ioutil.TempFile("", "cgroups")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// test update/load from PID
	manager = &ManagerV1{pid: pid}

	if err := manager.UpdateFromFile(tmpfile.Name()); err != nil {
		t.Fatal(err)
	}
	ensureIntInFile(t, cpuShares, 512)
}

func TestPauseResumeV1(t *testing.T) {
	test.EnsurePrivilege(t)
	require.CgroupsV1(t)

	manager := &ManagerV1{}
	if err := manager.Pause(); err == nil {
		t.Errorf("unexpected success with PID 0")
	}
	if err := manager.Resume(); err == nil {
		t.Errorf("unexpected success with PID 0")
	}

	cmd := exec.Command("/bin/cat", "/dev/zero")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	manager.pid = cmd.Process.Pid
	manager.path = filepath.Join("/singularity", strconv.Itoa(manager.pid))

	if err := manager.ApplyFromFile("example/cgroups.toml"); err != nil {
		t.Fatal(err)
	}
	defer manager.Remove()

	manager.Pause()
	// cgroups v1 freeze is to uninterruptible sleep
	ensureState(t, manager.pid, "D")

	manager.Resume()
	ensureState(t, manager.pid, "RS")
}

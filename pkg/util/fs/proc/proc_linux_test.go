// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package proc

import (
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestHasFilesystem(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	p, err := HasFilesystem("proc")
	if err != nil {
		t.Error(err)
	}
	if !p {
		t.Errorf("proc filesystem not present")
	}

	p, err = HasFilesystem("42fs")
	if err != nil {
		t.Error(err)
	}
	if p {
		t.Errorf("42fs should not be in supported filesystems")
	}
}

var mountInfoData = `22 28 0:21 / /sys rw,nosuid,nodev,noexec,relatime shared:7 - sysfs sysfs rw
23 28 0:4 / /proc rw,nosuid,nodev,noexec,relatime shared:13 - proc proc rw
24 28 0:6 / /dev rw,nosuid,relatime shared:2 - devtmpfs udev rw,size=8110616k,nr_inodes=2027654,mode=755
25 24 0:22 / /dev/pts rw,nosuid,noexec,relatime shared:3 - devpts devpts rw,gid=5,mode=620,ptmxmode=000
26 28 0:23 / /run rw,nosuid,noexec,relatime shared:5 - tmpfs tmpfs rw,size=1635872k,mode=755
28 0 8:1 / / rw,noatime,nodiratime shared:1 - ext4 /dev/sda1 rw,discard,errors=remount-ro,data=ordered
29 22 0:7 / /sys/kernel/security rw,nosuid,nodev,noexec,relatime shared:8 - securityfs securityfs rw
30 24 0:25 / /dev/shm rw,nosuid,nodev shared:4 - tmpfs tmpfs rw
31 26 0:26 / /run/lock rw,nosuid,nodev,noexec,relatime shared:6 - tmpfs tmpfs rw,size=5120k
32 22 0:27 / /sys/fs/cgroup ro,nosuid,nodev,noexec shared:9 - tmpfs tmpfs ro,mode=755
33 32 0:28 / /sys/fs/cgroup/unified rw,nosuid,nodev,noexec,relatime shared:10 - cgroup2 cgroup rw,nsdelegate
34 32 0:29 / /sys/fs/cgroup/systemd rw,nosuid,nodev,noexec,relatime shared:11 - cgroup cgroup rw,xattr,name=systemd
35 22 0:30 / /sys/fs/pstore rw,nosuid,nodev,noexec,relatime shared:12 - pstore pstore rw
36 32 0:31 / /sys/fs/cgroup/devices rw,nosuid,nodev,noexec,relatime shared:14 - cgroup cgroup rw,devices
37 32 0:32 / /sys/fs/cgroup/rdma rw,nosuid,nodev,noexec,relatime shared:15 - cgroup cgroup rw,rdma
38 32 0:33 / /sys/fs/cgroup/cpuset rw,nosuid,nodev,noexec,relatime shared:16 - cgroup cgroup rw,cpuset
39 32 0:34 / /sys/fs/cgroup/cpu,cpuacct rw,nosuid,nodev,noexec,relatime shared:17 - cgroup cgroup rw,cpu,cpuacct
40 32 0:35 / /sys/fs/cgroup/pids rw,nosuid,nodev,noexec,relatime shared:18 - cgroup cgroup rw,pids
41 32 0:36 / /sys/fs/cgroup/hugetlb rw,nosuid,nodev,noexec,relatime shared:19 - cgroup cgroup rw,hugetlb
42 32 0:37 / /sys/fs/cgroup/net_cls,net_prio rw,nosuid,nodev,noexec,relatime shared:20 - cgroup cgroup rw,net_cls,net_prio
43 32 0:38 / /sys/fs/cgroup/freezer rw,nosuid,nodev,noexec,relatime shared:21 - cgroup cgroup rw,freezer
44 32 0:39 / /sys/fs/cgroup/perf_event rw,nosuid,nodev,noexec,relatime shared:22 - cgroup cgroup rw,perf_event
45 32 0:40 / /sys/fs/cgroup/memory rw,nosuid,nodev,noexec,relatime shared:23 - cgroup cgroup rw,memory
46 32 0:41 / /sys/fs/cgroup/blkio rw,nosuid,nodev,noexec,relatime shared:24 - cgroup cgroup rw,blkio
47 23 0:42 / /proc/sys/fs/binfmt_misc rw,relatime shared:25 - autofs systemd-1 rw,fd=36,pgrp=1,timeout=0,minproto=5,maxproto=5,direct,pipe_ino=1423
48 24 0:19 / /dev/mqueue rw,relatime shared:26 - mqueue mqueue rw
49 24 0:43 / /dev/hugepages rw,relatime shared:27 - hugetlbfs hugetlbfs rw,pagesize=2M
50 22 0:8 / /sys/kernel/debug rw,relatime shared:28 - debugfs debugfs rw
51 26 0:44 / /run/rpc_pipefs rw,relatime shared:29 - rpc_pipefs sunrpc rw
52 22 0:20 / /sys/kernel/config rw,relatime shared:30 - configfs configfs rw
53 22 0:45 / /sys/fs/fuse/connections rw,relatime shared:31 - fusectl fusectl rw
54 23 0:46 / /proc/fs/nfsd rw,relatime shared:32 - nfsd nfsd rw
88 28 253:1 / /home rw,noatime,nodiratime shared:33 - ext4 /dev/mapper/pdc_egggdecf1 rw,errors=remount-ro,stripe=64
90 47 0:47 / /proc/sys/fs/binfmt_misc rw,relatime shared:34 - binfmt_misc binfmt_misc rw
381 26 0:54 / /run/user/1000 rw,nosuid,nodev,relatime shared:245 - tmpfs tmpfs rw,size=1635868k,mode=700,uid=1000,gid=1000
363 381 0:52 / /run/user/1000/gvfs rw,nosuid,nodev,relatime shared:233 - fuse.gvfsd-fuse gvfsd-fuse rw,user_id=1000,group_id=1000
579 28 0:65 / /tmp/squashfs rw,relatime - squashfs /dev/loop0 rw`

var expectedMap = map[string][]string{
	"/": {
		"/sys",
		"/proc",
		"/dev",
		"/run",
		"/home",
		"/tmp/squashfs",
	},
	"/dev": {
		"/dev/pts",
		"/dev/shm",
		"/dev/mqueue",
		"/dev/hugepages",
	},
	"/sys": {
		"/sys/kernel/security",
		"/sys/fs/cgroup",
		"/sys/fs/pstore",
		"/sys/kernel/debug",
		"/sys/kernel/config",
		"/sys/fs/fuse/connections",
	},
	"/sys/fs/cgroup": {
		"/sys/fs/cgroup/unified",
		"/sys/fs/cgroup/systemd",
		"/sys/fs/cgroup/devices",
		"/sys/fs/cgroup/rdma",
		"/sys/fs/cgroup/cpuset",
		"/sys/fs/cgroup/cpu,cpuacct",
		"/sys/fs/cgroup/pids",
		"/sys/fs/cgroup/hugetlb",
		"/sys/fs/cgroup/net_cls,net_prio",
		"/sys/fs/cgroup/freezer",
		"/sys/fs/cgroup/perf_event",
		"/sys/fs/cgroup/memory",
		"/sys/fs/cgroup/blkio",
	},
	"/proc": {
		"/proc/fs/nfsd",
		"/proc/sys/fs/binfmt_misc",
	},
	"/run": {
		"/run/lock",
		"/run/rpc_pipefs",
		"/run/user/1000",
	},
	"/run/user/1000": {
		"/run/user/1000/gvfs",
	},
}

func TestGetMountInfo(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	_, err := GetMountInfoEntry("/bad/path")
	if err == nil {
		t.Fatalf("unexpected success while parsing bad path")
	}

	tmpfile, err := ioutil.TempFile("", "mountinfo")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(mountInfoData)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	entries, err := GetMountInfoEntry(tmpfile.Name())
	if err != nil {
		t.Fatalf("unexpected error while parsing %s: %s", tmpfile.Name(), err)
	}

	for _, e := range entries {
		found := false
		if _, ok := expectedMap[e.Point]; ok {
			continue
		}
		for parent := range expectedMap {
			for _, child := range expectedMap[parent] {
				if e.Point == child {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("%s not found", e.Point)
		}
	}

	check := []MountInfoEntry{
		{
			ParentID:     "28",
			ID:           "22",
			Dev:          "0:21",
			Root:         "/",
			Fields:       "shared:7",
			FSType:       "sysfs",
			Source:       "sysfs",
			SuperOptions: []string{"rw"},
			Options:      []string{"rw", "nosuid", "nodev", "noexec", "relatime"},
		},
		{
			ParentID:     "28",
			ID:           "88",
			Dev:          "253:1",
			Root:         "/",
			Fields:       "shared:33",
			FSType:       "ext4",
			Source:       "/dev/mapper/pdc_egggdecf1",
			SuperOptions: []string{"rw", "errors=remount-ro", "stripe=64"},
			Options:      []string{"rw", "noatime", "nodiratime"},
		},
		{
			ParentID:     "28",
			ID:           "579",
			Dev:          "0:65",
			Root:         "/",
			Fields:       "",
			FSType:       "squashfs",
			Source:       "/dev/loop0",
			SuperOptions: []string{"rw"},
			Options:      []string{"rw", "relatime"},
		},
	}

	for _, c := range check {
		for _, e := range entries {
			if c.Point == e.Point {
				if e.ParentID != c.ParentID {
					t.Errorf("wrong parent ID %s instead of %s", e.ParentID, c.ParentID)
				}
				if e.ID != c.ID {
					t.Errorf("wrong ID: %s instead of %s", e.ID, c.ID)
				}
				if e.Dev != c.Dev {
					t.Errorf("wrong dev field: %s instead of %s", e.Dev, c.Dev)
				}
				if e.Root != c.Root {
					t.Errorf("wrong root field: %s instead of %s", e.Root, c.Root)
				}
				if e.Fields != c.Fields {
					t.Errorf("wrong fields: %s instead of %s", e.Fields, c.Fields)
				}
				if e.FSType != c.FSType {
					t.Errorf("wrong fstype: %s instead of %s", e.FSType, c.FSType)
				}
				if e.Source != c.Source {
					t.Errorf("wrong source: %s instead of %s", e.Source, c.Source)
				}
				if e.SuperOptions[0] != c.SuperOptions[0] {
					t.Errorf("wrong super options: %s instead of %s", e.SuperOptions[0], c.SuperOptions)
				}
				if e.Options[1] != c.Options[1] {
					t.Errorf("wrong options: %s instead of %s", e.Options[1], c.Options)
				}
			}
		}
	}
}

func TestGetMountPointMap(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	if _, err := GetMountPointMap("/proc/self/fakemountinfo"); err == nil {
		t.Errorf("should have failed with non existent path")
	}
	tmpfile, err := ioutil.TempFile("", "mountinfo")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(mountInfoData)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}
	m, err := GetMountPointMap(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if len(m) != len(expectedMap) {
		t.Errorf("got %d parent entry instead of %d", len(m), len(expectedMap))
	}

	for e := range expectedMap {
		if len(m[e]) != len(expectedMap[e]) {
			t.Errorf("got %d child mount point for %q instead of %d", len(m[e]), e, len(expectedMap[e]))
		}
		for _, c := range expectedMap[e] {
			found := false
			for _, mc := range m[e] {
				if mc == c {
					found = true
				}
			}
			if !found {
				t.Errorf("%s is missing for parent mount %s", c, e)
			}
		}
	}
}

func TestExtractPid(t *testing.T) {
	procList := []struct {
		path string
		pid  uint
		fail bool
	}{
		{"/proc/1/fd", 1, false},
		{"/proc/self", 0, true},
		{"/proc/123/ns/pid", 123, false},
		{"/proc/-1", 0, true},
		{"/etc/../proc/1/fd", 0, true},
	}
	for _, pl := range procList {
		pid, err := ExtractPid(pl.path)
		if err != nil && !pl.fail {
			t.Fatal(err)
		}
		if !pl.fail && pid != pl.pid {
			t.Fatalf("should have returned %d as PID instead of %d", pid, pl.pid)
		}
		if pl.fail && err == nil {
			t.Fatalf("extract path %s should have failed", pl.path)
		}
	}
}

func TestCountChilds(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	childs, err := CountChilds(1)
	if err != nil {
		t.Fatal(err)
	}
	if childs == 0 {
		t.Fatal("init have no child processes")
	}
	_, err = CountChilds(0)
	if err == nil {
		t.Fatal("no error reported with PID 0")
	}
}

func TestReadIDMap(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	_, _, err := ReadIDMap("/proc/__/uid_map")
	if err == nil {
		t.Fatalf("unexpected success with bad uid_map path")
	}

	// skip tests if uid_map doesn't exists
	if _, err := os.Stat("/proc/self/uid_map"); os.IsNotExist(err) {
		return
	}

	for _, e := range []string{"/proc/self/uid_map", "/proc/self/gid_map"} {
		containerID, hostID, err := ReadIDMap(e)
		if err != nil {
			t.Fatal(err)
		}
		if containerID != 0 || containerID != hostID {
			t.Errorf("unexpected container/host ID")
		}
	}

	for _, e := range []string{"a a a", "0 a a"} {
		f, err := ioutil.TempFile("", "uid_map-")
		if err != nil {
			t.Fatalf("failed to create temporary file")
		}
		defer os.Remove(f.Name())
		f.WriteString(e)
		f.Close()

		_, _, err = ReadIDMap(f.Name())
		if err == nil {
			t.Fatalf("unexpected success with bad formatted uid_map")
		}
	}
}

func TestParentMount(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	list := []struct {
		path   string
		parent string
		fail   bool
	}{
		{"/proc_", "", true},
		{"/proc", "/proc", false},
		{"/dev/null", "/dev", false},
		{"/proc/self", "/proc", false},
		{"/proc/fake", "", true},
	}

	for _, l := range list {
		parent, err := ParentMount(l.path)
		if l.fail && err == nil {
			t.Errorf("%s should fail", l.path)
		} else if !l.fail {
			if err != nil {
				t.Error(err)
			} else if parent != l.parent {
				t.Errorf("mount parent of %s should be %s not %s", l.path, l.parent, parent)
			}
		}

	}
}

func TestSetOOMScoreAdj(t *testing.T) {
	test.EnsurePrivilege(t)

	pid := os.Getpid()

	list := []struct {
		pid   int
		score int
		fail  bool
	}{
		{pid, 0, false},
		{pid, 10, false},
		{0, 0, true},
	}

	for _, l := range list {
		err := SetOOMScoreAdj(l.pid, &l.score)
		if l.fail && err == nil {
			t.Errorf("writing %d in /proc/%d/oom_score_adj should have failed", l.score, l.pid)
		} else if !l.fail && err != nil {
			t.Error(err)
		}
	}
}

func TestHasNamespace(t *testing.T) {
	test.EnsurePrivilege(t)

	has, err := HasNamespace(0, "ipc")
	if err == nil && has {
		t.Error("unexpected success with PID 0")
	}

	ppid := os.Getppid()
	has, err = HasNamespace(ppid, "net")
	if err != nil {
		t.Error(err)
	}
	if has {
		t.Errorf("namespaces should be identical")
	}

	cmd := exec.Command("/bin/cat")
	pipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Cloneflags = syscall.CLONE_NEWPID

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	has, err = HasNamespace(cmd.Process.Pid, "pid")
	if err != nil {
		t.Fatal(err)
	} else if !has {
		t.Errorf("pid namespace should be different")
	}

	pipe.Close()

	cmd.Wait()
}

func TestGetppid(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	pid := os.Getpid()
	ppid := os.Getppid()

	list := []struct {
		name          string
		pid           int
		ppid          int
		expectSuccess bool
	}{
		{"ProcessZero", 0, -1, false},
		{"CurrentProcess", pid, ppid, true},
		{"InitProcess", 1, -1, false},
	}

	for _, tt := range list {
		p, err := Getppid(tt.pid)
		if err != nil && tt.expectSuccess {
			t.Fatalf("unexpected failure for %q: %s", tt.name, err)
		} else if err == nil && !tt.expectSuccess {
			t.Fatalf("unexpected success for %q: got parent process ID %d instead of %d", tt.name, p, tt.ppid)
		} else if p != tt.ppid {
			t.Fatalf("unexpected parent process ID returned: got %d instead of %d", p, tt.ppid)
		}
	}
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
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
363 381 0:52 / /run/user/1000/gvfs rw,nosuid,nodev,relatime shared:233 - fuse.gvfsd-fuse gvfsd-fuse rw,user_id=1000,group_id=1000`

func TestParseMountInfo(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	if _, err := ParseMountInfo("/proc/self/fakemountinfo"); err == nil {
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
	m, err := ParseMountInfo(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 7 {
		t.Errorf("got %d main parent mount point instead of 7", len(m))
	}
	if len(m["/"]) != 5 {
		t.Errorf("got %d child mount point for '/' instead of 5", len(m["/"]))
	}
	if len(m["/dev"]) != 4 {
		t.Errorf("got %d child mount point for '/dev' instead of 4", len(m["/dev"]))
	}
	if len(m["/sys"]) != 6 {
		t.Errorf("got %d child mount point for '/sys' instead of 6", len(m["/sys"]))
	}
	if len(m["/sys/fs/cgroup"]) != 13 {
		t.Errorf("got %d child mount point for '/sys/fs/cgroup' instead of 13", len(m["/sys/fs/cgroup"]))
	}
	if len(m["/proc"]) != 2 {
		t.Errorf("got %d child mount point for '/proc' instead of 2", len(m["/proc"]))
	}
	if len(m["/run"]) != 3 {
		t.Errorf("got %d child mount point for '/run' instead of 3", len(m["/run"]))
	}
	if len(m["/run/user/1000"]) != 1 {
		t.Errorf("got %d child mount point for '/run/user/1000' instead of 1", len(m["/run/user/1000"]))
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
	childs, err = CountChilds(0)
	if err == nil {
		t.Fatal("no error reported with PID 0")
	}
}

func TestReadIDMap(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// skip tests if uid_map doesn't exists
	if _, err := os.Stat("/proc/self/uid_map"); os.IsNotExist(err) {
		return
	}
	containerID, hostID, err := ReadIDMap("/proc/self/uid_map")
	if err != nil {
		t.Fatal(err)
	}
	if containerID != 0 || containerID != hostID {
		t.Errorf("")
	}
	containerID, hostID, err = ReadIDMap("/proc/self/gid_map")
	if err != nil {
		t.Fatal(err)
	}
	if containerID != 0 || containerID != hostID {
		t.Errorf("")
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

	ppid := os.Getppid()
	has, err := HasNamespace(ppid, "net")
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
	if !has {
		t.Errorf("pid namespace should be different")
	}

	pipe.Close()

	cmd.Wait()
}

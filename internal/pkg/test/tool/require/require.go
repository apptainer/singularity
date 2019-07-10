// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package require

import (
	"os/exec"
	"sync"
	"syscall"
	"testing"

	"github.com/sylabs/singularity/pkg/util/fs/proc"
)

var hasUserNamespace bool
var hasUserNamespaceOnce sync.Once

// UserNamespace checks that the current test could use
// user namespace, if user namespaces are not enabled or
// supported, the current test is skipped with a message.
func UserNamespace(t *testing.T) {
	// not performance critical, just save extra execution
	// to get the same result
	hasUserNamespaceOnce.Do(func() {
		// user namespace is a bit special, as there is no simple
		// way to detect if it's supported or enabled via a call
		// on /proc/self/ns/user, the easiest and reliable way seems
		// to directly execute a command by requesting user namespace
		cmd := exec.Command("/bin/true")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUSER,
		}
		// no error means user namespaces are enabled
		hasUserNamespace = cmd.Run() == nil
	})
	if !hasUserNamespace {
		t.Skipf("user namespaces seems not enabled or supported")
	}
}

// Filesystem checks that the current test could use the
// corresponding filesystem, if the filesystem is not
// listed in /proc/filesystems, the current test is skipped
// with a message.
func Filesystem(t *testing.T, fs string) {
	has, err := proc.HasFilesystem(fs)
	if err != nil {
		t.Fatalf("error while checking filesystem presence: %s", err)
	}
	if !has {
		t.Skipf("%s filesystem seems not supported", fs)
	}
}

// CgroupMount checks that /sys/fs/cgroup mount is present,
// if no cgroup mount is found, the current test is skipped
// with a message.
func CgroupMount(t *testing.T) {
	// check first that cgroup is enabled
	Filesystem(t, "cgroup")

	mounts, err := proc.ParseMountInfo("/proc/self/mountinfo")
	if err != nil {
		t.Fatalf("could not obtain mount information: %s", err)
	}
	// checks if /sys/fs/cgroup is mounted
	_, hasCgroupMount := mounts["/sys/fs/cgroup"]
	if !hasCgroupMount {
		t.Skipf("no /sys/fs/cgroup mount found")
	}
}

// Command checks if the provided command is found
// in one the path defined in the PATH environment variable,
// if not found the current test is skipped with a message.
func Command(t *testing.T, command string) {
	_, err := exec.LookPath(command)
	if err != nil {
		t.Skipf("%s command not found in $PATH", command)
	}
}

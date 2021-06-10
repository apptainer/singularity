// Copyright (c) 2019-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package require

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/containerd/cgroups"
	"github.com/hpcng/singularity/internal/pkg/buildcfg"
	"github.com/hpcng/singularity/internal/pkg/security/seccomp"
	"github.com/hpcng/singularity/pkg/network"
	"github.com/hpcng/singularity/pkg/util/fs/proc"
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
		err := cmd.Run()
		hasUserNamespace = err == nil
		if !hasUserNamespace {
			t.Logf("Could not use user namespaces: %s", err)
		}
	})
	if !hasUserNamespace {
		t.Skipf("user namespaces seems not enabled or supported")
	}
}

var supportNetwork bool
var supportNetworkOnce sync.Once

// Network check that bridge network is supported by
// system, if not the current test is skipped with a
// message.
func Network(t *testing.T) {
	supportNetworkOnce.Do(func() {
		logFn := func(err error) {
			t.Logf("Could not use network: %s", err)
		}

		ctx := context.TODO()

		cmd := exec.Command("/bin/cat")
		cmd.SysProcAttr = &syscall.SysProcAttr{}
		cmd.SysProcAttr.Cloneflags = syscall.CLONE_NEWNET

		stdinPipe, err := cmd.StdinPipe()
		if err != nil {
			logFn(err)
			return
		}

		err = cmd.Start()
		if err != nil {
			logFn(err)
			return
		}

		nsPath := fmt.Sprintf("/proc/%d/ns/net", cmd.Process.Pid)

		cniPath := new(network.CNIPath)
		cniPath.Conf = filepath.Join(buildcfg.SYSCONFDIR, "singularity", "network")
		cniPath.Plugin = filepath.Join(buildcfg.LIBEXECDIR, "singularity", "cni")

		setup, err := network.NewSetup([]string{"bridge"}, "_test_", nsPath, cniPath)
		if err != nil {
			logFn(err)
			return
		}
		if err := setup.AddNetworks(ctx); err != nil {
			logFn(err)
			return
		}
		if err := setup.DelNetworks(ctx); err != nil {
			logFn(err)
			return
		}

		stdinPipe.Close()

		if err := cmd.Wait(); err != nil {
			logFn(err)
			return
		}

		supportNetwork = true
	})
	if !supportNetwork {
		t.Skipf("Network (bridge) not supported")
	}
}

// Cgroups checks that cgroups is enabled, if not the
// current test is skipped with a message.
func Cgroups(t *testing.T) {
	_, err := cgroups.V1()
	if err != nil {
		t.Skipf("cgroups disabled")
	}
}

// CgroupsFreezer checks that cgroup freezer subsystem is
// available, if not the current test is skipped with a
// message
func CgroupsFreezer(t *testing.T) {
	subSys, err := cgroups.V1()
	if err != nil {
		t.Skipf("cgroups disabled")
	}
	for _, s := range subSys {
		if s.Name() == "freezer" {
			return
		}
	}
	t.Skipf("no cgroups freezer subsystem available")
}

// Nvidia checks that an NVIDIA stack is available
func Nvidia(t *testing.T) {
	nvsmi, err := exec.LookPath("nvidia-smi")
	if err != nil {
		t.Skipf("nvidia-smi not found on PATH: %v", err)
	}
	cmd := exec.Command(nvsmi)
	if err := cmd.Run(); err != nil {
		t.Skipf("nvidia-smi failed to run: %v", err)
	}
}

// Rocm checks that a Rocm stack is available
func Rocm(t *testing.T) {
	rocminfo, err := exec.LookPath("rocminfo")
	if err != nil {
		t.Skipf("rocminfo not found on PATH: %v", err)
	}
	cmd := exec.Command(rocminfo)
	if output, err := cmd.Output(); err != nil {
		t.Skipf("rocminfo failed to run: %v - %v", err, string(output))
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

// Command checks if the provided command is found
// in one the path defined in the PATH environment variable,
// if not found the current test is skipped with a message.
func Command(t *testing.T, command string) {
	_, err := exec.LookPath(command)
	if err != nil {
		t.Skipf("%s command not found in $PATH", command)
	}
}

// Seccomp checks that seccomp is enabled, if not the
// current test is skipped with a message.
func Seccomp(t *testing.T) {
	if !seccomp.Enabled() {
		t.Skipf("seccomp disabled, Singularity was compiled without the seccomp library")
	}
}

// Arch checks the test machine has the specified architecture.
// If not, the test is skipped with a message.
func Arch(t *testing.T, arch string) {
	if arch != "" && runtime.GOARCH != arch {
		t.Skipf("test requires architecture %s", arch)
	}

}

// ArchIn checks the test machine is one of the specified archs.
// If not, the test is skipped with a message.
func ArchIn(t *testing.T, archs []string) {
	if len(archs) > 0 {
		b := runtime.GOARCH
		for _, a := range archs {
			if b == a {
				return
			}
		}
		t.Skipf("test requires architecture %s", strings.Join(archs, "|"))
	}
}

// MkfsExt3 checks that mkfs.ext3 is available and
// support -d option to create writable overlay layout.
func MkfsExt3(t *testing.T) {
	mkfs, err := exec.LookPath("mkfs.ext3")
	if err != nil {
		t.Skipf("mkfs.ext3 not found in $PATH")
	}

	buf := new(bytes.Buffer)
	cmd := exec.Command(mkfs, "--help")
	cmd.Stderr = buf
	_ = cmd.Run()

	if !strings.Contains(buf.String(), "[-d ") {
		t.Skipf("mkfs.ext3 is too old and doesn't support -d")
	}
}

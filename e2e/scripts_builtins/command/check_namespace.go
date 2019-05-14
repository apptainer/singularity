// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package command

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/sylabs/singularity/pkg/stest"
	"mvdan.cc/sh/v3/interp"
)

var nsSupported = make(map[string]bool)

// check-namespace builtin
// usage:
// check-namespace <namespace>
func checkNamespace(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("check-namespace requires as argument: user, mnt, net, uts, ipc, pid or cgroup")
	}
	if supported, has := nsSupported[args[0]]; has {
		if !supported {
			return interp.ExitStatus(1)
		}
		return interp.ExitStatus(0)
	}

	switch args[0] {
	case "user":
		uid := int(os.Getuid())
		gid := int(os.Getgid())

		cmd := exec.Command("/bin/sh", "-c", "true")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUSER,
			UidMappings: []syscall.SysProcIDMap{
				{
					ContainerID: uid,
					HostID:      uid,
					Size:        1,
				},
			},
			GidMappings: []syscall.SysProcIDMap{
				{
					ContainerID: gid,
					HostID:      gid,
					Size:        1,
				},
			},
		}
		if err := cmd.Run(); err != nil {
			nsSupported[args[0]] = false
			return interp.ExitStatus(1)
		}
		nsSupported[args[0]] = true
	case "mnt", "net", "uts", "pid", "ipc", "cgroup":
		path := filepath.Join("/proc/self/ns", args[0])
		if _, err := os.Stat(path); os.IsNotExist(err) {
			nsSupported[args[0]] = false
			return interp.ExitStatus(1)
		}
		nsSupported[args[0]] = true
	default:
		return fmt.Errorf("unknown namespace %q", args[0])
	}

	return interp.ExitStatus(0)
}

func init() {
	stest.RegisterCommandBuiltin("check-namespace", checkNamespace)
}

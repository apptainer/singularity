// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package namespaces

import (
	"os/exec"
	"syscall"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestEnter(t *testing.T) {
	test.EnsurePrivilege(t)

	cmd := exec.Command("/bin/cat")
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Cloneflags = syscall.CLONE_NEWIPC | syscall.CLONE_NEWNET

	pipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	if err := Enter(cmd.Process.Pid, "ipc"); err != nil {
		t.Error(err)
	}
	if err := Enter(cmd.Process.Pid, "net"); err != nil {
		t.Error(err)
	}

	pipe.Close()

	if err := cmd.Wait(); err != nil {
		t.Error(err)
	}

	if err := Enter(0, "net"); err == nil {
		t.Errorf("should have failed with bad process")
	}
	if err := Enter(cmd.Process.Pid, "user"); err == nil {
		t.Error("should have failed with unsupported namespace")
	}
}

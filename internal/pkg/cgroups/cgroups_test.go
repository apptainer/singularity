// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cgroups

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestPutConfig(t *testing.T) {
	test.EnsurePrivilege(t)

	cmd := exec.Command("/bin/cat")
	pipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	pid := cmd.Process.Pid
	strPid := strconv.Itoa(pid)
	path := filepath.Join("/singularity", strPid)

	manager := &Manager{Pid: pid, Path: path}
	if err := manager.ApplyFromFile("example/cgroups.toml"); err != nil {
		t.Errorf("%s", err)
	}

	pipe.Close()

	cmd.Wait()

	if err := manager.Remove(); err != nil {
		t.Errorf("%s", err)
	}
}

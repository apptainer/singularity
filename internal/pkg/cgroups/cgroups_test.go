// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cgroups

import (
	"os"
	"strconv"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestPutConfig(t *testing.T) {
	test.EnsurePrivilege(t)

	pid := os.Getpid()
	strPid := strconv.Itoa(pid)

	manager := &Manager{Pid: pid, Name: strPid}
	if err := manager.ApplyFromFile("example/cgroups.toml"); err != nil {
		t.Errorf("%s", err)
	}

	if err := manager.Remove(); err != nil {
		t.Errorf("%s", err)
	}
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package mount

import (
	"fmt"
	"syscall"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestSystem(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	points := &Points{}

	points.AddBind(BindsTag, "/etc/hosts", "/etc/hosts", syscall.MS_BIND|syscall.MS_REC)

	system := &System{
		Points: points,
	}

	before := false
	after := false
	mnt := false

	mountFn := func(point *Point, system *System) error {
		mnt = true
		return nil
	}
	beforeHook := func(system *System) error {
		before = true
		tag := system.CurrentTag()
		if tag != BindsTag {
			return fmt.Errorf("bad tag returned: %s instead of %s", tag, BindsTag)
		}
		return nil
	}
	afterHook := func(system *System) error {
		after = true
		if system.Mount == nil {
			system.Mount = mountFn
		}
		tag := system.CurrentTag()
		if tag != BindsTag {
			return fmt.Errorf("bad tag returned: %s instead of %s", tag, BindsTag)
		}
		return nil
	}

	if err := system.RunBeforeTag("fakeTag", beforeHook); err == nil {
		t.Errorf("RunBeforeTag should have failed with unauthorized tag")
	}
	if err := system.RunAfterTag("fakeTag", afterHook); err == nil {
		t.Errorf("RunAfterTag should have failed with unauthorized tag")
	}
	if err := system.RunBeforeTag(BindsTag, beforeHook); err != nil {
		t.Error(err)
	}
	if err := system.RunAfterTag(BindsTag, afterHook); err != nil {
		t.Error(err)
	}
	if err := system.MountAll(); err != nil {
		t.Error(err)
	}
	if before == false {
		t.Errorf("beforeHook wasn't executed")
	}
	if after == false {
		t.Errorf("afterHook wasn't executed")
	}
	if err := system.MountAll(); err != nil {
		t.Error(err)
	}
	if mnt == false {
		t.Errorf("mountFn wasn't executed")
	}
}

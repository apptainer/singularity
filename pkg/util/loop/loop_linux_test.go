// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package loop

import (
	"fmt"
	"os"
	"syscall"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestLoop(t *testing.T) {
	test.EnsurePrivilege(t)

	var i1 *Info64

	info := &Info64{
		Flags: FlagsAutoClear | FlagsReadOnly,
	}
	loopDevOne := &Device{
		MaxLoopDevices: GetMaxLoopDevices(),
		Info:           info,
	}
	loopDevTwo := &Device{
		MaxLoopDevices: GetMaxLoopDevices(),
		Info:           info,
	}

	loopOne := -1
	loopTwo := -1

	// With wrong path and file pointer
	if err := loopDevOne.AttachFromPath("", os.O_RDONLY, &loopOne); err == nil {
		t.Errorf("unexpected success with a wrong path")
	}
	if err := loopDevOne.AttachFromFile(nil, os.O_RDONLY, &loopOne); err == nil {
		t.Errorf("unexpected success with a nil file pointer")
	}

	// With good file
	if err := loopDevOne.AttachFromPath("/etc/passwd", os.O_RDONLY, &loopOne); err != nil {
		t.Error(err)
	}
	defer loopDevOne.Close()

	f, err := os.Open("/etc/passwd")
	if err != nil {
		t.Error(err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		t.Error(err)
	}

	// With correct file pointer
	if err := loopDevTwo.AttachFromFile(f, os.O_RDONLY, &loopTwo); err != nil {
		t.Error(err)
	}
	if loopOne == loopTwo {
		t.Errorf("attached to the same loop block device /dev/loop%d", loopOne)
	}
	// Test if loop devices matches associated file
	_, err = GetStatusFromPath("")
	if err == nil {
		t.Errorf("unexpected success while returning status with non existent loop device")
	}

	path := fmt.Sprintf("/dev/loop%d", loopTwo)
	i1, err = GetStatusFromPath(path)
	if err != nil {
		t.Error(err)
	}

	loopDevTwo.Close()

	st := fi.Sys().(*syscall.Stat_t)
	// cast to uint64 as st.Dev is uint32 on MIPS
	if uint64(st.Dev) != i1.Device || st.Ino != i1.Inode {
		t.Errorf("bad file association for %s", path)
	}

	// With shared loop device
	loopDevTwo.Shared = true
	loopTwo = -1
	if err := loopDevTwo.AttachFromPath("/etc/passwd", os.O_RDONLY, &loopTwo); err != nil {
		t.Error(err)
	}
	loopDevTwo.Close()

	if loopOne != loopTwo {
		t.Errorf("not attached to the same loop block device /dev/loop%d", loopOne)
	}

	loopTwo = -1
	if err := loopDevTwo.AttachFromPath("/etc/group", os.O_RDONLY, &loopTwo); err != nil {
		t.Error(err)
	}
	loopDevTwo.Close()

	if loopOne == loopTwo {
		t.Errorf("attached to the same loop block device /dev/loop%d", loopOne)
	}

	// With MaxLoopDevices set to zero
	loopDevTwo.MaxLoopDevices = 0
	if err := loopDevTwo.AttachFromPath("/etc/group", os.O_RDONLY, &loopTwo); err == nil {
		t.Errorf("unexpected success with MaxLoopDevices = 0")
	}
}

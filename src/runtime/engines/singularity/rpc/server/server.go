// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package server

import (
	"fmt"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/loop"
	args "github.com/singularityware/singularity/src/runtime/engines/singularity/rpc"
)

// Methods is a receiver type
type Methods int

// Mount performs a mount with the specified arguments
func (t *Methods) Mount(arguments *args.MountArgs, reply *int) error {
	return syscall.Mount(arguments.Source, arguments.Target, arguments.Filesystem, arguments.Mountflags, arguments.Data)
}

// Mkdir performs a mkdir with the specified arguments
func (t *Methods) Mkdir(arguments *args.MkdirArgs, reply *int) error {
	fmt.Println("Mkdir requested")
	return nil
}

// Chroot performs a chroot with the specified arguments
func (t *Methods) Chroot(arguments *args.ChrootArgs, reply *int) error {
	if err := syscall.Chdir(arguments.Root); err != nil {
		return fmt.Errorf("Failed to change directory to %s", arguments.Root)
	}

	sylog.Debugf("Called pivot_root(%s, etc)\n", arguments.Root)
	if err := syscall.PivotRoot(".", "etc"); err != nil {
		return fmt.Errorf("pivot_root %s: %s", arguments.Root, err)
	}

	sylog.Debugf("Called chroot(%s)\n", arguments.Root)
	if err := syscall.Chroot("."); err != nil {
		return fmt.Errorf("chroot %s", err)
	}

	sylog.Debugf("Called unmount(etc, syscall.MNT_DETACH)\n")
	if err := syscall.Unmount("etc", syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount pivot_root dir %s", err)
	}

	sylog.Debugf("Changing directory to / to avoid getpwd issues\n")
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir / %s", err)
	}
	return nil
}

// LoopDevice attaches a loop device with the specified arguments
func (t *Methods) LoopDevice(arguments *args.LoopArgs, reply *int) error {
	loopdev := new(loop.Device)

	if err := loopdev.Attach(arguments.Image, arguments.Mode, reply); err != nil {
		return err
	}
	if err := loopdev.SetStatus(&arguments.Info); err != nil {
		return err
	}
	return nil
}

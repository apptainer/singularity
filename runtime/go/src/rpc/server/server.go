/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package server

import (
	"fmt"
	"loop"
	args "rpc"
	"syscall"
)

type RpcOps int

func (t *RpcOps) Mount(arguments *args.MountArgs, reply *int) error {
	return syscall.Mount(arguments.Source, arguments.Target, arguments.Filesystem, arguments.Mountflags, arguments.Data)
}

func (t *RpcOps) Mkdir(arguments *args.MkdirArgs, reply *int) error {
	fmt.Println("Mkdir requested")
	return nil
}

func (t *RpcOps) Chroot(arguments *args.ChrootArgs, reply *int) error {
	if err := syscall.Chdir(arguments.Root); err != nil {
		return fmt.Errorf("Failed to change directory to %s", arguments.Root)
	}
	if err := syscall.PivotRoot(".", "etc"); err != nil {
		return fmt.Errorf("pivot_root %s: %s", arguments.Root, err)
	}
	if err := syscall.Chroot("."); err != nil {
		return fmt.Errorf("chroot %s", err)
	}
	if err := syscall.Unmount("etc", syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount pivot_root dir %s", err)
	}
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir / %s", err)
	}
	return nil
}

func (t *RpcOps) LoopDevice(arguments *args.LoopArgs, reply *int) error {
	loopdev := new(loop.LoopDevice)

	if err := loopdev.Attach(arguments.Image, arguments.Mode, reply); err != nil {
		return err
	}
	if err := loopdev.SetStatus(&arguments.Info); err != nil {
		return err
	}
	return nil
}

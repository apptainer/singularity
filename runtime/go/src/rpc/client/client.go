/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package client

import (
	"loop"
	"net/rpc"
	args "rpc"
)

type RpcOps struct {
	Client *rpc.Client
}

func (t *RpcOps) Mount(source string, target string, filesystem string, flags uintptr, data string) (int, error) {
	arguments := &args.MountArgs{source, target, filesystem, flags, data}
	var reply int
	err := t.Client.Call("Privileged.Mount", arguments, &reply)
	return reply, err
}

func (t *RpcOps) Mkdir(path string) (int, error) {
	arguments := &args.MkdirArgs{path}
	var reply int
	err := t.Client.Call("Privileged.Mkdir", arguments, &reply)
	return reply, err
}

func (t *RpcOps) Chroot(root string) (int, error) {
	arguments := &args.ChrootArgs{root}
	var reply int
	err := t.Client.Call("Privileged.Chroot", arguments, &reply)
	return reply, err
}

func (t *RpcOps) LoopDevice(image string, mode int, info loop.LoopInfo64) (int, error) {
	arguments := &args.LoopArgs{image, mode, info}
	var reply int
	err := t.Client.Call("Privileged.LoopDevice", arguments, &reply)
	return reply, err
}

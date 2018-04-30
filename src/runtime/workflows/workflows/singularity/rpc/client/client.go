/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package client

import (
	"net/rpc"

	"github.com/singularityware/singularity/src/pkg/util/loop"
	args "github.com/singularityware/singularity/src/runtime/workflows/workflows/singularity/rpc"
)

type Rpc struct {
	Client *rpc.Client
	Name   string
}

func (t *Rpc) Mount(source string, target string, filesystem string, flags uintptr, data string) (int, error) {
	arguments := &args.MountArgs{Source: source, Target: target, Filesystem: filesystem, Mountflags: flags, Data: data}
	var reply int
	err := t.Client.Call(t.Name+".Mount", arguments, &reply)
	return reply, err
}

func (t *Rpc) Mkdir(path string) (int, error) {
	arguments := &args.MkdirArgs{Path: path}
	var reply int
	err := t.Client.Call(t.Name+".Mkdir", arguments, &reply)
	return reply, err
}

func (t *Rpc) Chroot(root string) (int, error) {
	arguments := &args.ChrootArgs{Root: root}
	var reply int
	err := t.Client.Call(t.Name+".Chroot", arguments, &reply)
	return reply, err
}

func (t *Rpc) LoopDevice(image string, mode int, info loop.LoopInfo64) (int, error) {
	arguments := &args.LoopArgs{Image: image, Mode: mode, Info: info}
	var reply int
	err := t.Client.Call(t.Name+".LoopDevice", arguments, &reply)
	return reply, err
}

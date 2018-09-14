// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"net/rpc"

	"github.com/singularityware/singularity/src/pkg/util/loop"
	args "github.com/singularityware/singularity/src/runtime/engines/singularity/rpc"
)

// RPC holds the state necessary for remote procedure calls
type RPC struct {
	Client *rpc.Client
	Name   string
}

// Mount calls tme mount RPC using the supplied arguments
func (t *RPC) Mount(source string, target string, filesystem string, flags uintptr, data string) (int, error) {
	arguments := &args.MountArgs{
		Source:     source,
		Target:     target,
		Filesystem: filesystem,
		Mountflags: flags,
		Data:       data,
	}
	var reply int
	err := t.Client.Call(t.Name+".Mount", arguments, &reply)
	return reply, err
}

// Mkdir calls the mkdir RPC using the supplied arguments
func (t *RPC) Mkdir(path string) (int, error) {
	arguments := &args.MkdirArgs{
		Path: path,
	}
	var reply int
	err := t.Client.Call(t.Name+".Mkdir", arguments, &reply)
	return reply, err
}

// Chroot calls the chroot RPC using the supplied arguments
func (t *RPC) Chroot(root string) (int, error) {
	arguments := &args.ChrootArgs{
		Root: root,
	}
	var reply int
	err := t.Client.Call(t.Name+".Chroot", arguments, &reply)
	return reply, err
}

// LoopDevice calls the loop device RPC using the supplied arguments
func (t *RPC) LoopDevice(image string, mode int, info loop.Info64) (int, error) {
	arguments := &args.LoopArgs{
		Image: image,
		Mode:  mode,
		Info:  info,
	}
	var reply int
	err := t.Client.Call(t.Name+".LoopDevice", arguments, &reply)
	return reply, err
}

// SetHostname calls the sethostname RPC using the supplied arguments
func (t *RPC) SetHostname(hostname string) (int, error) {
	arguments := &args.HostnameArgs{
		Hostname: hostname,
	}
	var reply int
	err := t.Client.Call(t.Name+".SetHostname", arguments, &reply)
	return reply, err
}

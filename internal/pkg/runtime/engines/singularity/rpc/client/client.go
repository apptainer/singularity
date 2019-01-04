// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"net/rpc"
	"os"

	args "github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/rpc"
	"github.com/sylabs/singularity/pkg/util/loop"
)

// RPC holds the state necessary for remote procedure calls.
type RPC struct {
	Client *rpc.Client
	Name   string
}

// Mount calls tme mount RPC using the supplied arguments.
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

// Mkdir calls the mkdir RPC using the supplied arguments.
func (t *RPC) Mkdir(path string, perm os.FileMode) (int, error) {
	arguments := &args.MkdirArgs{
		Path: path,
		Perm: perm,
	}
	var reply int
	err := t.Client.Call(t.Name+".Mkdir", arguments, &reply)
	return reply, err
}

// Chroot calls the chroot RPC using the supplied arguments.
func (t *RPC) Chroot(root string, method string) (int, error) {
	arguments := &args.ChrootArgs{
		Root:   root,
		Method: method,
	}
	var reply int
	err := t.Client.Call(t.Name+".Chroot", arguments, &reply)
	return reply, err
}

// LoopDevice calls the loop device RPC using the supplied arguments.
func (t *RPC) LoopDevice(image string, mode int, info loop.Info64, maxDevices int, shared bool) (int, error) {
	arguments := &args.LoopArgs{
		Image:      image,
		Mode:       mode,
		Info:       info,
		MaxDevices: maxDevices,
		Shared:     shared,
	}
	var reply int
	err := t.Client.Call(t.Name+".LoopDevice", arguments, &reply)
	return reply, err
}

// SetHostname calls the sethostname RPC using the supplied arguments.
func (t *RPC) SetHostname(hostname string) (int, error) {
	arguments := &args.HostnameArgs{
		Hostname: hostname,
	}
	var reply int
	err := t.Client.Call(t.Name+".SetHostname", arguments, &reply)
	return reply, err
}

// HasNamespace calls the HasNamespace RPC using the supplied arguments.
func (t *RPC) HasNamespace(pid int, nstype string) (bool, error) {
	arguments := &args.HasNamespaceArgs{
		Pid:    pid,
		NsType: nstype,
	}
	var reply int
	err := t.Client.Call(t.Name+".HasNamespace", arguments, &reply)
	if err != nil {
		return false, err
	}
	if reply == 1 {
		return true, err
	}
	return false, err
}

// SetFsID calls the setfsid RPC using the supplied arguments.
func (t *RPC) SetFsID(uid int, gid int) (int, error) {
	arguments := &args.SetFsIDArgs{
		UID: uid,
		GID: gid,
	}
	var reply int
	err := t.Client.Call(t.Name+".SetFsID", arguments, &reply)
	return reply, err
}

// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"encoding/gob"
	"net/rpc"
	"os"
	"syscall"

	args "github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/rpc"
	"github.com/sylabs/singularity/pkg/util/loop"
)

// RPC holds the state necessary for remote procedure calls.
type RPC struct {
	Client *rpc.Client
	Name   string
}

// Mount calls the mount RPC using the supplied arguments.
func (t *RPC) Mount(source string, target string, filesystem string, flags uintptr, data string) error {
	arguments := &args.MountArgs{
		Source:     source,
		Target:     target,
		Filesystem: filesystem,
		Mountflags: flags,
		Data:       data,
	}

	var mountErr error

	err := t.Client.Call(t.Name+".Mount", arguments, &mountErr)
	// RPC communication will take precedence over mount error
	if err == nil {
		err = mountErr
	}

	return err
}

// Decrypt calls the DeCrypt RPC using the supplied arguments.
func (t *RPC) Decrypt(offset uint64, path string, key []byte, masterPid int) (string, error) {
	arguments := &args.CryptArgs{
		Offset:    offset,
		Loopdev:   path,
		Key:       key,
		MasterPid: masterPid,
	}

	var reply string
	err := t.Client.Call(t.Name+".Decrypt", arguments, &reply)

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

// Chdir calls the chdir RPC using the supplied arguments.
func (t *RPC) Chdir(dir string) (int, error) {
	arguments := &args.ChdirArgs{
		Dir: dir,
	}
	var reply int
	err := t.Client.Call(t.Name+".Chdir", arguments, &reply)
	return reply, err
}

func init() {
	var sysErrnoType syscall.Errno
	// register syscall.Errno as a type we need to get back
	gob.Register(sysErrnoType)
}

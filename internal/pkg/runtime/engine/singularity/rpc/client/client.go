// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"net/rpc"
	"os"

	args "github.com/sylabs/singularity/internal/pkg/runtime/engine/singularity/rpc"
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
func (t *RPC) Mkdir(path string, perm os.FileMode) error {
	arguments := &args.MkdirArgs{
		Path: path,
		Perm: perm,
	}
	return t.Client.Call(t.Name+".Mkdir", arguments, nil)
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

// Chdir calls the chdir RPC using the supplied arguments.
func (t *RPC) Chdir(dir string) (int, error) {
	arguments := &args.ChdirArgs{
		Dir: dir,
	}
	var reply int
	err := t.Client.Call(t.Name+".Chdir", arguments, &reply)
	return reply, err
}

// Stat calls the stat RPC using the supplied arguments.
func (t *RPC) Stat(path string) (os.FileInfo, error) {
	arguments := &args.StatArgs{
		Path: path,
	}
	var reply args.StatReply
	err := t.Client.Call(t.Name+".Stat", arguments, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Fi, reply.Err
}

// Lstat calls the lstat RPC using the supplied arguments.
func (t *RPC) Lstat(path string) (os.FileInfo, error) {
	arguments := &args.StatArgs{
		Path: path,
	}
	var reply args.StatReply
	err := t.Client.Call(t.Name+".Lstat", arguments, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Fi, reply.Err
}

// SendFuseFd calls the SendFuseFd RPC using the supplied arguments.
func (t *RPC) SendFuseFd(socket int, fds []int) error {
	arguments := &args.SendFuseFdArgs{
		Socket: socket,
		Fds:    fds,
	}
	var reply int
	err := t.Client.Call(t.Name+".SendFuseFd", arguments, &reply)
	return err
}

// OpenSendFuseFd calls the OpenSendFuseFd RPC using the supplied arguments.
func (t *RPC) OpenSendFuseFd(socket int) (int, error) {
	arguments := &args.OpenSendFuseFdArgs{
		Socket: socket,
	}
	var reply int
	err := t.Client.Call(t.Name+".OpenSendFuseFd", arguments, &reply)
	return reply, err
}

// Symlink calls the mkdir RPC using the supplied arguments.
func (t *RPC) Symlink(old string, new string) error {
	arguments := &args.SymlinkArgs{
		Old: old,
		New: new,
	}
	return t.Client.Call(t.Name+".Symlink", arguments, nil)
}

// ReadDir calls the readdir RPC using the supplied arguments.
func (t *RPC) ReadDir(dir string) ([]os.FileInfo, error) {
	arguments := &args.ReadDirArgs{
		Dir: dir,
	}
	var reply args.ReadDirReply
	err := t.Client.Call(t.Name+".ReadDir", arguments, &reply)
	return reply.Files, err
}

// Chown calls the chown RPC using the supplied arguments.
func (t *RPC) Chown(name string, uid int, gid int) error {
	arguments := &args.ChownArgs{
		Name: name,
		UID:  uid,
		GID:  gid,
	}
	return t.Client.Call(t.Name+".Chown", arguments, nil)
}

// Lchown calls the lchown RPC using the supplied arguments.
func (t *RPC) Lchown(name string, uid int, gid int) error {
	arguments := &args.ChownArgs{
		Name: name,
		UID:  uid,
		GID:  gid,
	}
	return t.Client.Call(t.Name+".Lchown", arguments, nil)
}

// EvalRelative calls the evalrelative RPC using the supplied arguments.
func (t *RPC) EvalRelative(name string, root string) string {
	arguments := &args.EvalRelativeArgs{
		Name: name,
		Root: root,
	}
	var reply string
	t.Client.Call(t.Name+".EvalRelative", arguments, &reply)
	return reply
}

// Lchown calls the lchown RPC using the supplied arguments.
func (t *RPC) Readlink(name string) (string, error) {
	arguments := &args.ReadlinkArgs{
		Name: name,
	}
	var reply string
	err := t.Client.Call(t.Name+".Readlink", arguments, &reply)
	return reply, err
}

// Umask calls the umask RPC using the supplied arguments.
func (t *RPC) Umask(mask int) int {
	arguments := &args.UmaskArgs{
		Mask: mask,
	}
	var reply int
	t.Client.Call(t.Name+".Umask", arguments, &reply)
	return reply
}

// WriteFile calls the writefile RPC using the supplied arguments.
func (t *RPC) WriteFile(filename string, data []byte, perm os.FileMode) error {
	arguments := &args.WriteFileArgs{
		Filename: filename,
		Data:     data,
		Perm:     perm,
	}
	return t.Client.Call(t.Name+".WriteFile", arguments, nil)
}

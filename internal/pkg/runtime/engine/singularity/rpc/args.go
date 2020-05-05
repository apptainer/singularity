// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package rpc

import (
	"encoding/gob"
	"os"
	"syscall"
	"time"

	"github.com/sylabs/singularity/pkg/util/loop"
)

// MkdirArgs defines the arguments to mkdir.
type MkdirArgs struct {
	Path string
	Perm os.FileMode
}

// LoopArgs defines the arguments to create a loop device.
type LoopArgs struct {
	Image      string
	Mode       int
	Info       loop.Info64
	MaxDevices int
	Shared     bool
}

// MountArgs defines the arguments to mount.
type MountArgs struct {
	Source     string
	Target     string
	Filesystem string
	Mountflags uintptr
	Data       string
}

// CryptArgs defines the arguments to mount.
type CryptArgs struct {
	Offset    uint64
	Loopdev   string
	Key       []byte
	MasterPid int
}

// ChrootArgs defines the arguments to chroot.
type ChrootArgs struct {
	Root   string
	Method string
}

// HostnameArgs defines the arguments to sethostname.
type HostnameArgs struct {
	Hostname string
}

// ChdirArgs defines the arguments to chdir.
type ChdirArgs struct {
	Dir string
}

// StatReply defines the reply for stat.
type StatReply struct {
	Fi  os.FileInfo
	Err error
}

// StatArgs defines the arguments to stat.
type StatArgs struct {
	Path string
}

// SendFuseFdArgs defines the arguments to send fuse file descriptor.
type SendFuseFdArgs struct {
	Socket int
	Fds    []int
}

// OpenFuseFdArgs defines the arguments to open and send a fuse file descriptor.
type OpenSendFuseFdArgs struct {
	Socket int
}

// SymlinkArgs defines the arguments to symlink.
type SymlinkArgs struct {
	Old string
	New string
}

// ReadDirArgs defines the arguments to readdir.
type ReadDirArgs struct {
	Dir string
}

// ReadDirReply defines the reply for readdir.
type ReadDirReply struct {
	Files []os.FileInfo
}

// ChownArgs defines the arguments to chown/lchown.
type ChownArgs struct {
	Name string
	UID  int
	GID  int
}

// EvalRelativeArgs defines the arguments to evalrelative.
type EvalRelativeArgs struct {
	Name string
	Root string
}

// ReadlinkArgs defines the arguments to readlink.
type ReadlinkArgs struct {
	Name string
}

// UmaskArgs defines the arguments to umask.
type UmaskArgs struct {
	Mask int
}

// WriteFileArgs defines the arguments to writefile.
type WriteFileArgs struct {
	Filename string
	Data     []byte
	Perm     os.FileMode
}

// FileInfo returns FileInfo interface to be passed as RPC argument.
func FileInfo(fi os.FileInfo) os.FileInfo {
	return &fileInfo{
		N:  fi.Name(),
		S:  fi.Size(),
		M:  fi.Mode(),
		T:  fi.ModTime(),
		Sy: fi.Sys(),
	}
}

// fileInfo internal interface with exported fields.
type fileInfo struct {
	N  string
	S  int64
	M  os.FileMode
	T  time.Time
	Sy interface{}
}

func (fi fileInfo) Name() string {
	return fi.N
}

func (fi fileInfo) Size() int64 {
	return fi.S
}

func (fi fileInfo) Mode() os.FileMode {
	return fi.M
}

func (fi fileInfo) ModTime() time.Time {
	if fi.T.IsZero() {
		return time.Now()
	}
	return fi.T
}

func (fi fileInfo) IsDir() bool {
	return fi.M.IsDir()
}

func (fi fileInfo) Sys() interface{} {
	return fi.Sy
}

func init() {
	gob.Register(syscall.Errno(0))
	gob.Register((*fileInfo)(nil))
	gob.Register((*syscall.Stat_t)(nil))
	gob.Register((*os.PathError)(nil))
	gob.Register((*os.SyscallError)(nil))
	gob.Register((*os.LinkError)(nil))
}

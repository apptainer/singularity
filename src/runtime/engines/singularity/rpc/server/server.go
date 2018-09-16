// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package server

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/util/mainthread"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/loop"
	args "github.com/singularityware/singularity/src/runtime/engines/singularity/rpc"
)

// Methods is a receiver type
type Methods int

// Mount performs a mount with the specified arguments
func (t *Methods) Mount(arguments *args.MountArgs, reply *int) (err error) {
	mainthread.Execute(func() {
		err = syscall.Mount(arguments.Source, arguments.Target, arguments.Filesystem, arguments.Mountflags, arguments.Data)
	})
	return err
}

// Mkdir performs a mkdir with the specified arguments
func (t *Methods) Mkdir(arguments *args.MkdirArgs, reply *int) (err error) {
	mainthread.Execute(func() {
		oldmask := syscall.Umask(0)
		err = os.Mkdir(arguments.Path, arguments.Perm)
		syscall.Umask(oldmask)
	})
	return err
}

// Chroot performs a chroot with the specified arguments
func (t *Methods) Chroot(arguments *args.ChrootArgs, reply *int) error {
	// idea taken from libcontainer (and also LXC developpers) to avoid
	// creation of temporary directory or use of existing directory
	// for pivot_root

	sylog.Debugf("Hold reference to host / directory")
	oldroot, err := os.Open("/")
	if err != nil {
		return fmt.Errorf("failed to open host root directory: %s", err)
	}
	defer oldroot.Close()

	sylog.Debugf("Change current directory to %s", arguments.Root)
	if err := syscall.Chdir(arguments.Root); err != nil {
		return fmt.Errorf("failed to change directory to %s", arguments.Root)
	}

	sylog.Debugf("Called pivot_root on %s\n", arguments.Root)
	if err := syscall.PivotRoot(".", "."); err != nil {
		return fmt.Errorf("pivot_root %s: %s", arguments.Root, err)
	}

	sylog.Debugf("Change current directory to host / directory")
	if err := syscall.Fchdir(int(oldroot.Fd())); err != nil {
		return fmt.Errorf("failed to change directory to old root: %s", err)
	}

	sylog.Debugf("Apply slave mount propagation for host / directory")
	if err := syscall.Mount("", ".", "", syscall.MS_SLAVE|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to apply slave mount propagation for host / directory: %s", err)
	}

	sylog.Debugf("Called unmount(/, syscall.MNT_DETACH)\n")
	if err := syscall.Unmount(".", syscall.MNT_DETACH); err != nil {
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
	var image *os.File
	loopdev := new(loop.Device)

	if strings.HasPrefix(arguments.Image, "/proc/self/fd/") {
		strFd := strings.TrimPrefix(arguments.Image, "/proc/self/fd/")
		fd, err := strconv.ParseUint(strFd, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to convert image file descriptor: %s", err)
		}
		image = os.NewFile(uintptr(fd), "")
		if err != nil {
			return fmt.Errorf("can't find image %s", arguments.Image)
		}
	} else {
		var err error

		image, err = os.OpenFile(arguments.Image, arguments.Mode, 0600)
		if err != nil {
			return err
		}
	}

	runtime.LockOSThread()
	syscall.Setfsuid(0)

	defer runtime.UnlockOSThread()
	defer syscall.Setfsuid(os.Getuid())

	if err := loopdev.AttachFromFile(image, arguments.Mode, reply); err != nil {
		return err
	}
	if err := loopdev.SetStatus(&arguments.Info); err != nil {
		return err
	}
	return nil
}

// SetHostname sets hostname with the specified arguments
func (t *Methods) SetHostname(arguments *args.HostnameArgs, reply *int) error {
	return syscall.Sethostname([]byte(arguments.Hostname))
}

// HasNamespace checks if host namespace and container namespace
// are different and sets reply to 0 or 1
func (t *Methods) HasNamespace(arguments *args.HasNamespaceArgs, reply *int) error {
	var st1 syscall.Stat_t
	var st2 syscall.Stat_t

	processOne := fmt.Sprintf("/proc/1/ns/%s", arguments.NsType)
	processTwo := fmt.Sprintf("/proc/self/ns/%s", arguments.NsType)

	if err := syscall.Stat(processOne, &st1); err != nil {
		return err
	}
	if err := syscall.Stat(processTwo, &st2); err != nil {
		return err
	}

	if st1.Ino != st2.Ino {
		*reply = 1
	} else {
		*reply = 0
	}

	return nil
}

// SetFsID sets filesystem uid and gid
func (t *Methods) SetFsID(arguments *args.SetFsIDArgs, reply *int) error {
	mainthread.Execute(func() {
		syscall.Setfsuid(arguments.UID)
		syscall.Setfsgid(arguments.GID)
	})
	return nil
}

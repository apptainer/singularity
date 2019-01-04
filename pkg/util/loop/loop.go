// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package loop

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/util/fs/lock"
)

// Device describes a loop device
type Device struct {
	MaxLoopDevices int
	Shared         bool
	Info           *Info64
	file           *os.File
}

// AttachFromFile finds a free loop device, opens it, and stores file descriptor
// provided by image file pointer
func (loop *Device) AttachFromFile(image *os.File, mode int, number *int) error {
	var path string

	fi, err := image.Stat()
	if err != nil {
		return err
	}
	st := fi.Sys().(*syscall.Stat_t)
	imageIno := st.Ino
	imageDev := st.Dev

	fd, err := lock.Exclusive("/dev")
	if err != nil {
		return err
	}
	defer lock.Release(fd)

	for device := 0; device <= loop.MaxLoopDevices; device++ {
		*number = device

		path = fmt.Sprintf("/dev/loop%d", device)
		if fi, err := os.Stat(path); err != nil {
			dev := int((7 << 8) | (device & 0xff) | ((device & 0xfff00) << 12))
			esys := syscall.Mknod(path, syscall.S_IFBLK|0660, dev)
			if errno, ok := esys.(syscall.Errno); ok {
				if errno != syscall.EEXIST {
					return esys
				}
			}
		} else if fi.Mode()&os.ModeDevice == 0 {
			return fmt.Errorf("%s is not a block device", path)
		}

		if loop.file, err = os.OpenFile(path, mode, 0600); err != nil {
			continue
		}
		if loop.Shared {
			status, err := GetStatusFromFile(loop.file)
			loop.file.Close()
			if err != nil {
				return err
			}
			if status.Inode == imageIno && status.Device == imageDev &&
				status.Flags&FlagsReadOnly == loop.Info.Flags&FlagsReadOnly &&
				status.Offset == loop.Info.Offset && status.SizeLimit == loop.Info.SizeLimit {
				sylog.Debugf("Found shared loop device /dev/loop%d", device)
				return nil
			}
			if *number == loop.MaxLoopDevices {
				loop.Shared = false
				device = 0
			}
		} else {
			_, _, esys := syscall.Syscall(syscall.SYS_IOCTL, loop.file.Fd(), CmdSetFd, image.Fd())
			if esys != 0 {
				loop.file.Close()
				continue
			}
			break
		}
	}

	if *number == loop.MaxLoopDevices {
		return fmt.Errorf("no loop devices available")
	}

	if _, _, err := syscall.Syscall(syscall.SYS_FCNTL, loop.file.Fd(), syscall.F_SETFD, syscall.FD_CLOEXEC); err != 0 {
		return fmt.Errorf("failed to set close-on-exec on loop device %s: %s", path, err.Error())
	}

	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, loop.file.Fd(), CmdSetStatus64, uintptr(unsafe.Pointer(loop.Info))); err != 0 {
		return fmt.Errorf("Failed to set loop flags on loop device: %s", syscall.Errno(err))
	}

	return nil
}

// AttachFromPath finds a free loop device, opens it, and stores file descriptor
// of opened image path
func (loop *Device) AttachFromPath(image string, mode int, number *int) error {
	file, err := os.OpenFile(image, mode, 0600)
	if err != nil {
		return err
	}
	return loop.AttachFromFile(file, mode, number)
}

// GetStatusFromFile gets info status about an opened loop device
func GetStatusFromFile(loop *os.File) (*Info64, error) {
	info := &Info64{}
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, loop.Fd(), CmdGetStatus64, uintptr(unsafe.Pointer(info)))
	if err != syscall.ENXIO && err != 0 {
		return nil, fmt.Errorf("Failed to get loop flags for loop device: %s", err.Error())
	}
	return info, nil
}

// GetStatusFromPath gets info status about a loop device from path
func GetStatusFromPath(path string) (*Info64, error) {
	loop, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open loop device %s: %s", path, err)
	}
	return GetStatusFromFile(loop)
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package loop

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/fs/lock"
)

// Device describes a loop device
type Device struct {
	loop           *os.File
	MaxLoopDevices uint
	Shared         bool
}

// Attach finds a free loop device, opens it, and stores file descriptor
func (d *Device) Attach(image string, info *Info64, number *int) error {
	var path string
	var imgIno uint64
	var imgDev uint64
	matchedDevice := -1
	freeDevice := -1

	if d.MaxLoopDevices > 256 {
		d.MaxLoopDevices = 256
	}
	mode := os.O_RDONLY
	if info.Flags&FlagsReadOnly == 0 {
		mode = os.O_RDWR
	}

	img, err := os.OpenFile(image, mode, 0)
	if err != nil {
		return err
	}
	defer img.Close()

	fi, err := img.Stat()
	if err != nil {
		return err
	}
	st := fi.Sys().(*syscall.Stat_t)
	imgIno = st.Ino
	imgDev = st.Dev

	runtime.LockOSThread()

	defer runtime.UnlockOSThread()
	defer syscall.Setfsuid(os.Getuid())

	syscall.Setfsuid(0)

	fd, err := lock.Exclusive("/dev")
	if err != nil {
		return err
	}
	defer lock.Release(fd)

	for device := 0; device < int(d.MaxLoopDevices); device++ {
		path = fmt.Sprintf("/dev/loop%d", device)
		d.loop, err = os.OpenFile(path, mode, 0)
		if err != nil {
			dev := int((7 << 8) | device)
			err := syscall.Mknod(path, syscall.S_IFBLK|0600, dev)
			if err != nil {
				return err
			}
			if err := syscall.Chown(path, 0, 0); err != nil {
				return err
			}
			d.loop, err = os.OpenFile(path, mode, 0)
			if err != nil {
				continue
			}
		}
		status, err := d.GetStatus()
		if err != nil {
			d.loop.Close()
			return err
		}
		if freeDevice == -1 && status.Inode == 0 {
			freeDevice = device
			if !d.Shared {
				d.loop.Close()
				break
			}
		}
		if d.Shared && status.Inode == imgIno && status.Device == imgDev &&
			status.Flags&FlagsReadOnly == info.Flags&FlagsReadOnly &&
			status.Offset == info.Offset && status.SizeLimit == info.SizeLimit {
			matchedDevice = device
			break
		}
		d.loop.Close()
	}

	if matchedDevice != -1 {
		sylog.Debugf("Found shared loop device /dev/loop%d", freeDevice)
		*number = matchedDevice
		return nil
	} else if freeDevice != -1 {
		path = fmt.Sprintf("/dev/loop%d", freeDevice)
		sylog.Debugf("Opening loop device %s", path)
		d.loop, err = os.OpenFile(path, mode, 0)
		if err != nil {
			return err
		}
		if err := d.SetFd(img.Fd()); err == nil {
			if err := d.SetStatus(info); err != nil {
				return err
			}
		} else {
			sylog.Debugf("Could not associate image to loop: %s", err)
			d.loop.Close()
		}
		if _, _, err := syscall.Syscall(syscall.SYS_FCNTL, d.loop.Fd(), syscall.F_SETFD, syscall.FD_CLOEXEC); err != 0 {
			return fmt.Errorf("failed to set close-on-exec on loop device %s: %s", path, err.Error())
		}
		*number = freeDevice
		return nil
	}

	return errors.New("No loop devices available")
}

// SetStatus sets info status about image
func (d *Device) SetStatus(info *Info64) error {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, d.loop.Fd(), CmdSetStatus64, uintptr(unsafe.Pointer(info)))
	if err != 0 {
		return fmt.Errorf("Failed to set loop flags on loop device: %s", err.Error())
	}
	return nil
}

// GetStatus gets info status about opened loop device
func (d *Device) GetStatus() (*Info64, error) {
	info := &Info64{}
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, d.loop.Fd(), CmdGetStatus64, uintptr(unsafe.Pointer(info)))
	if err != syscall.ENXIO && err != 0 {
		return nil, fmt.Errorf("Failed to get loop flags for loop device: %s", err.Error())
	}
	return info, nil
}

// SetFd associates loop device with image identified by its file descriptor
func (d *Device) SetFd(fd uintptr) error {
	_, _, esys := syscall.Syscall(syscall.SYS_IOCTL, d.loop.Fd(), CmdSetFd, fd)
	if esys != 0 {
		return syscall.Errno(esys)
	}
	return nil
}

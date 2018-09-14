// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package loop

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

// MaxLoopDevices is the maxiumum number of loop devices that are supported
const MaxLoopDevices = 256

// Device describes a loop device
type Device struct {
	file *os.File
}

// Attach finds a free loop device, opens it, and stores file descriptor
func (loop *Device) Attach(image string, mode int, number *int) error {
	var path string

	runtime.LockOSThread()

	defer runtime.UnlockOSThread()
	defer syscall.Setfsuid(os.Getuid())

	img, err := os.OpenFile(image, mode, 0600)
	if err != nil {
		return err
	}

	syscall.Setfsuid(0)

	for device := 0; device < MaxLoopDevices; device++ {
		path = fmt.Sprintf("/dev/loop%d", device)
		if fi, err := os.Stat(path); err != nil {
			dev := int((7 << 8) | device)
			esys := syscall.Mknod(path, syscall.S_IFBLK|0600, dev)
			if esys != nil {
				return esys
			}
		} else {
			if (fi.Mode() & os.ModeDevice) == 0 {
				return fmt.Errorf("%s is not a block device", path)
			}
		}
		file, err := os.OpenFile(path, mode, 0600)
		if err != nil {
			continue
		}
		_, _, esys := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), CmdSetFd, img.Fd())
		if esys != 0 {
			file.Close()
			continue
		}
		if device == MaxLoopDevices {
			break
		}
		loop.file = file
		*number = device
		return nil
	}

	return errors.New("No loop devices available")
}

// SetStatus sets info status about image
func (loop *Device) SetStatus(info *Info64) error {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, loop.file.Fd(), CmdSetStatus64, uintptr(unsafe.Pointer(info)))
	if err != 0 {
		return fmt.Errorf("Failed to set loop flags on loop device: %s", syscall.Errno(err))
	}
	return nil
}

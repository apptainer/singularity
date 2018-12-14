// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package loop

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// Device describes a loop device
type Device struct {
	MaxLoopDevices int
	file           *os.File
}

// AttachFromFile finds a free loop device, opens it, and stores file descriptor
// provided by image file pointer
func (loop *Device) AttachFromFile(image *os.File, mode int, number *int) error {
	var path string

	for device := 0; device < loop.MaxLoopDevices; device++ {
		path = fmt.Sprintf("/dev/loop%d", device)
		if fi, err := os.Stat(path); err != nil {
			dev := int((7 << 8) | device)
			esys := syscall.Mknod(path, syscall.S_IFBLK|0660, dev)
			if errno, ok := esys.(syscall.Errno); ok {
				if errno != syscall.EEXIST {
					return esys
				}
			}
		} else if fi.Mode()&os.ModeDevice == 0 {
			return fmt.Errorf("%s is not a block device", path)
		}

		loopDev, err := os.OpenFile(path, mode, 0600)
		if err != nil {
			continue
		}
		_, _, esys := syscall.Syscall(syscall.SYS_IOCTL, loopDev.Fd(), CmdSetFd, image.Fd())
		if esys != 0 {
			loopDev.Close()
			continue
		}
		if device == loop.MaxLoopDevices {
			break
		}
		loop.file = loopDev
		*number = device

		if _, _, err := syscall.Syscall(syscall.SYS_FCNTL, loopDev.Fd(), syscall.F_SETFD, syscall.FD_CLOEXEC); err != 0 {
			return fmt.Errorf("failed to set close-on-exec on loop device %s: %s", path, err.Error())
		}

		return nil
	}

	return errors.New("No loop devices available")
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

// SetStatus sets info status about image
func (loop *Device) SetStatus(info *Info64) error {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, loop.file.Fd(), CmdSetStatus64, uintptr(unsafe.Pointer(info)))
	if err != 0 {
		return fmt.Errorf("Failed to set loop flags on loop device: %s", syscall.Errno(err))
	}
	return nil
}

// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package loop

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

// AttachFromFile finds a free loop device, opens it, and stores file descriptor
// provided by image file pointer
func (loop *Device) AttachFromFile(image *os.File, mode int, number *int) error {
	var path string
	var loopCtlFd int
	var loopFd int

	if image == nil {
		return fmt.Errorf("empty file pointer")
	}

	_, err := image.Stat()
	if err != nil {
		return err
	}

	if loopCtlFd, err = syscall.Open("/dev/loop-control", os.O_RDONLY, 0600); err != nil {
		return err
	}
	defer syscall.Close(loopCtlFd)

	for {
		// get a new loop device using LOOP_CTL_GET_FREE
		n, _, esys := syscall.Syscall(syscall.SYS_IOCTL, uintptr(loopCtlFd), CmdGetFree, 0)
		if esys != 0 {
			return err
		}

		*number = int(n)

		if *number > loop.MaxLoopDevices {
			return fmt.Errorf("dynamically allocated loop device exceeds maximum")
		}

		path = fmt.Sprintf("/dev/loop%d", n)

		// try to open the loop device
		if loopFd, err = syscall.Open(path, mode, 0600); err != nil {
			// Failed to open the device node; the node should've been created
			// automatically but maybe we got here before that happened. We
			// restart the loop and look for another free loop device using
			// LOOP_CTL_GET_FREE; if the device hasn't been taken yet we will
			// get the same number and hopefully this time the node got
			// created correctly. Note that we wait a little bit to avoid
			// a tight loop with high cpu usage.
			time.Sleep(10 * time.Millisecond)
			continue
		}

		// attach the image file to the loop device
		_, _, esys = syscall.Syscall(syscall.SYS_IOCTL, uintptr(loopFd), CmdSetFd, image.Fd())
		if esys != 0 {
			// Failed to attach the image. Most likely we lost the race and
			// some other process took the loop device before we got here. We
			// restart the loop and look for another free loop device using
			// LOOP_CTL_GET_FREE.
			syscall.Close(loopFd)
			continue
		}

		break
	}

	if _, _, err := syscall.Syscall(syscall.SYS_FCNTL, uintptr(loopFd), syscall.F_SETFD, syscall.FD_CLOEXEC); err != 0 {
		return fmt.Errorf("failed to set close-on-exec on loop device %s: %s", path, err.Error())
	}

	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(loopFd), CmdSetStatus64, uintptr(unsafe.Pointer(loop.Info))); err != 0 {
			if err == syscall.EAGAIN && i < maxRetries-1 {
				// with changes introduces in https://github.com/torvalds/linux/commit/5db470e229e22b7eda6e23b5566e532c96fb5bc3
				// loop_set_status() can temporarily fail with EAGAIN -> sleep and try again
				// (cf. https://github.com/karelzak/util-linux/blob/dab1303287b7ebe30b57ccc78591070dad0a85ea/lib/loopdev.c#L1355)
				time.Sleep(250 * time.Millisecond)
				continue
			}
			// clear associated file descriptor to release the loop device,
			// best-effort here without error checking because we need the
			// error from previous ioctl call
			syscall.Syscall(syscall.SYS_IOCTL, uintptr(loopFd), CmdClrFd, 0)
			return fmt.Errorf("failed to set loop flags on loop device: %s", syscall.Errno(err))
		}
		break
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

// GetStatusFromFd gets info status about an opened loop device
func GetStatusFromFd(fd uintptr) (*Info64, error) {
	info := &Info64{}
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, CmdGetStatus64, uintptr(unsafe.Pointer(info)))
	if err != syscall.ENXIO && err != 0 {
		return nil, fmt.Errorf("failed to get loop flags for loop device: %s", err.Error())
	}
	return info, nil
}

// GetStatusFromPath gets info status about a loop device from path
func GetStatusFromPath(path string) (*Info64, error) {
	loop, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open loop device %s: %s", path, err)
	}
	return GetStatusFromFd(loop.Fd())
}

// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
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

	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/hpcng/singularity/pkg/util/fs/lock"
)

// Device describes a loop device
type Device struct {
	MaxLoopDevices int
	Shared         bool
	Info           *Info64
	fd             *int
}

// Loop device flags values
const (
	FlagsReadOnly  = 1
	FlagsAutoClear = 4
	FlagsPartScan  = 8
	FlagsDirectIO  = 16
)

// Loop device encryption types
const (
	CryptNone      = 0
	CryptXor       = 1
	CryptDes       = 2
	CryptFish2     = 3
	CryptBlow      = 4
	CryptCast128   = 5
	CryptIdea      = 6
	CryptDummy     = 9
	CryptSkipJack  = 10
	CryptCryptoAPI = 18
	CryptMax       = 20
)

// Loop device IOCTL commands
const (
	CmdSetFd       = 0x4C00
	CmdClrFd       = 0x4C01
	CmdSetStatus   = 0x4C02
	CmdGetStatus   = 0x4C03
	CmdSetStatus64 = 0x4C04
	CmdGetStatus64 = 0x4C05
	CmdChangeFd    = 0x4C06
	CmdSetCapacity = 0x4C07
	CmdSetDirectIO = 0x4C08
)

// Info64 contains information about a loop device.
type Info64 struct {
	Device         uint64
	Inode          uint64
	Rdevice        uint64
	Offset         uint64
	SizeLimit      uint64
	Number         uint32
	EncryptType    uint32
	EncryptKeySize uint32
	Flags          uint32
	FileName       [64]byte
	CryptName      [64]byte
	EncryptKey     [32]byte
	Init           [2]uint64
}

// AttachFromFile finds a free loop device, opens it, and stores file descriptor
// provided by image file pointer
func (loop *Device) AttachFromFile(image *os.File, mode int, number *int) error {
	var path string
	var loopFd int

	if image == nil {
		return fmt.Errorf("empty file pointer")
	}

	fi, err := image.Stat()
	if err != nil {
		return err
	}
	st := fi.Sys().(*syscall.Stat_t)
	imageIno := st.Ino
	// cast to uint64 as st.Dev is uint32 on MIPS
	imageDev := uint64(st.Dev)

	fd, err := lock.Exclusive("/dev")
	if err != nil {
		return err
	}
	defer lock.Release(fd)

	freeDevice := -1

	for device := 0; device <= loop.MaxLoopDevices; device++ {
		*number = device

		if device == loop.MaxLoopDevices {
			if loop.Shared {
				loop.Shared = false
				if freeDevice != -1 {
					device = freeDevice
					continue
				}
			}
			return fmt.Errorf("no loop devices available")
		}

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

		if loopFd, err = syscall.Open(path, mode, 0600); err != nil {
			continue
		}
		if loop.Shared {
			status, err := GetStatusFromFd(uintptr(loopFd))
			if err != nil {
				syscall.Close(loopFd)
				sylog.Debugf("Could not get loop device %d status: %s", device, err)
				continue
			}
			// there is no associated image with loop device, save indice so second loop
			// iteration will start from this device
			if status.Inode == 0 && freeDevice == -1 {
				freeDevice = device
				syscall.Close(loopFd)
				continue
			}
			if status.Inode == imageIno && status.Device == imageDev &&
				status.Flags&FlagsReadOnly == loop.Info.Flags&FlagsReadOnly &&
				status.Offset == loop.Info.Offset && status.SizeLimit == loop.Info.SizeLimit {
				// keep the reference to the loop device file descriptor to
				// be sure that the loop device won't be released between this
				// check and the mount of the filesystem
				sylog.Debugf("Sharing loop device %d", device)
				loop.fd = new(int)
				*loop.fd = loopFd
				return nil
			}
			syscall.Close(loopFd)
		} else {
			_, _, esys := syscall.Syscall(syscall.SYS_IOCTL, uintptr(loopFd), CmdSetFd, image.Fd())
			if esys != 0 {
				syscall.Close(loopFd)
				continue
			}
			break
		}
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

	loop.fd = new(int)
	*loop.fd = loopFd
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

// Close closes the loop device.
func (loop *Device) Close() error {
	if loop.fd != nil {
		return syscall.Close(*loop.fd)
	}
	return nil
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

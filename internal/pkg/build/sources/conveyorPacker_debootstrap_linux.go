// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

// prepareFakerootEnv prepares a build environment to
// make fakeroot working with debootstrap.
func (cp *DebootstrapConveyorPacker) prepareFakerootEnv(ctx context.Context) (func(), error) {
	truePath, err := exec.LookPath("true")
	if err != nil {
		return nil, fmt.Errorf("while searching true command: %s", err)
	}
	mountPath, err := exec.LookPath("mount")
	if err != nil {
		return nil, fmt.Errorf("while searching mount command: %s", err)
	}
	mknodPath, err := exec.LookPath("mknod")
	if err != nil {
		return nil, fmt.Errorf("while searching mknod command: %s", err)
	}

	procFsPath := "/proc/filesystems"

	devs := []string{
		"/dev/null",
		"/dev/random",
		"/dev/urandom",
		"/dev/zero",
	}

	devPath := filepath.Join(cp.b.RootfsPath, "dev")
	if err := os.Mkdir(devPath, 0755); err != nil {
		return nil, fmt.Errorf("while creating %s: %s", devPath, err)
	}

	innerCtx, cancel := context.WithCancel(ctx)

	umountFn := func() {
		cancel()

		syscall.Unmount(mountPath, syscall.MNT_DETACH)
		syscall.Unmount(mknodPath, syscall.MNT_DETACH)
		for _, d := range devs {
			path := filepath.Join(cp.b.RootfsPath, d)
			syscall.Unmount(path, syscall.MNT_DETACH)
		}
	}

	// bind /bin/true on top of mount/mknod command
	// so debootstrap wouldn't fail while preparing
	// chroot environment
	if err := syscall.Mount(truePath, mountPath, "", syscall.MS_BIND, ""); err != nil {
		return umountFn, fmt.Errorf("while mounting %s to %s: %s", truePath, mountPath, err)
	}
	if err := syscall.Mount(truePath, mknodPath, "", syscall.MS_BIND, ""); err != nil {
		return umountFn, fmt.Errorf("while mounting %s to %s: %s", truePath, mknodPath, err)
	}

	// very dirty workaround to address issue with makedev
	// package installation during ubuntu bootstrap, we watch
	// for the creation of $ROOTFS/sbin/MAKEDEV and truncate
	// the file to obtain an equivalent of /bin/true, for makedev
	// post-configuration package we also need to create at least
	// one /dev/ttyX file
	go func() {
		makedevPath := filepath.Join(cp.b.RootfsPath, "/sbin/MAKEDEV")
		for {
			select {
			case <-innerCtx.Done():
				break
			case <-time.After(100 * time.Millisecond):
				if _, err := os.Stat(makedevPath); err == nil {
					os.Truncate(makedevPath, 0)
					os.Create(filepath.Join(cp.b.RootfsPath, "/dev/tty1"))
					break
				}
			}
		}
	}()

	// debootstrap look at /proc/filesystems to check
	// if sysfs is present, we bind /dev/null on top
	// of /proc/filesystems to trick debootstrap to not
	// mount /sys
	if err := syscall.Mount("/dev/null", procFsPath, "", syscall.MS_BIND, ""); err != nil {
		return umountFn, fmt.Errorf("while mounting /dev/null to %s: %s", procFsPath, err)
	}

	// mount required block devices
	for _, p := range devs {
		rootfsPath := filepath.Join(cp.b.RootfsPath, p)
		if err := fs.Touch(rootfsPath); err != nil {
			return umountFn, fmt.Errorf("while creating %s: %s", rootfsPath, err)
		}
		if err := syscall.Mount(p, rootfsPath, "", syscall.MS_BIND, ""); err != nil {
			return umountFn, fmt.Errorf("while mounting %s to %s: %s", p, rootfsPath, err)
		}
	}

	return umountFn, nil
}

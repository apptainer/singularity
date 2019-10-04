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

	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

// prepareFakerootEnv prepares a build environment to
// make fakeroot working with pacstrap.
func (cp *ArchConveyorPacker) prepareFakerootEnv(ctx context.Context) (func(), error) {
	truePath, err := exec.LookPath("true")
	if err != nil {
		return nil, fmt.Errorf("while searching true command: %s", err)
	}
	mountPath, err := exec.LookPath("mount")
	if err != nil {
		return nil, fmt.Errorf("while searching mount command: %s", err)
	}
	umountPath, err := exec.LookPath("umount")
	if err != nil {
		return nil, fmt.Errorf("while searching umount command: %s", err)
	}

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
	procPath := filepath.Join(cp.b.RootfsPath, "proc")
	if err := os.Mkdir(procPath, 0755); err != nil {
		return nil, fmt.Errorf("while creating %s: %s", procPath, err)
	}

	umountFn := func() {
		syscall.Unmount(mountPath, syscall.MNT_DETACH)
		syscall.Unmount(umountPath, syscall.MNT_DETACH)
		syscall.Unmount(procPath, syscall.MNT_DETACH)
		for _, d := range devs {
			path := filepath.Join(cp.b.RootfsPath, d)
			syscall.Unmount(path, syscall.MNT_DETACH)
		}
	}

	// bind /bin/true on top of mount/umount command so
	// pacstrap wouldn't fail while preparing chroot
	// environment
	if err := syscall.Mount(truePath, mountPath, "", syscall.MS_BIND, ""); err != nil {
		return umountFn, fmt.Errorf("while mounting %s to %s: %s", truePath, mountPath, err)
	}
	if err := syscall.Mount(truePath, umountPath, "", syscall.MS_BIND, ""); err != nil {
		return umountFn, fmt.Errorf("while mounting %s to %s: %s", truePath, umountPath, err)
	}
	if err := syscall.Mount("/proc", procPath, "", syscall.MS_BIND, ""); err != nil {
		return umountFn, fmt.Errorf("while mounting /proc to %s: %s", procPath, err)
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

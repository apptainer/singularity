// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/hpcng/singularity/internal/pkg/util/fs"
	"github.com/hpcng/singularity/pkg/build/types"
	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/hpcng/singularity/pkg/util/namespaces"
)

const (
	pacmanConfURL = "https://github.com/archlinux/svntogit-packages/raw/master/pacman/trunk/pacman.conf"
)

var (
	// Default list of packages to install when bootstrapping arch
	// As of 2019-10-06 there is a base metapackage instead of a base group
	// https://www.archlinux.org/news/base-group-replaced-by-mandatory-base-package-manual-intervention-required/
	instList = []string{"base"}
)

// ArchConveyorPacker only needs to hold the conveyor to have the needed data to pack
type ArchConveyorPacker struct {
	b *types.Bundle
}

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

// Get just stores the source
func (cp *ArchConveyorPacker) Get(ctx context.Context, b *types.Bundle) (err error) {
	cp.b = b

	//check for pacstrap on system
	pacstrapPath, err := exec.LookPath("pacstrap")
	if err != nil {
		return fmt.Errorf("pacstrap is not in PATH: %v", err)
	}

	//make sure architecture is supported
	if arch := runtime.GOARCH; arch != `amd64` {
		return fmt.Errorf("%v architecture is not supported", arch)
	}

	pacConf, err := cp.getPacConf(pacmanConfURL)
	if err != nil {
		return fmt.Errorf("while getting pacman config: %v", err)
	}

	insideUserNs, setgroupsAllowed := namespaces.IsInsideUserNamespace(os.Getpid())
	if insideUserNs && setgroupsAllowed {
		umountFn, err := cp.prepareFakerootEnv(ctx)
		if umountFn != nil {
			defer umountFn()
		}
		if err != nil {
			return fmt.Errorf("while preparing fakeroot build environment: %s", err)
		}
	}

	args := []string{"-C", pacConf, "-c", "-d", "-G", "-M", cp.b.RootfsPath, "haveged"}
	args = append(args, instList...)

	pacCmd := exec.Command(pacstrapPath, args...)
	pacCmd.Stdout = os.Stdout
	pacCmd.Stderr = os.Stderr
	sylog.Debugf("\n\tPacstrap Path: %s\n\tPac Conf: %s\n\tRootfs: %s\n\tInstall List: %s\n", pacstrapPath, pacConf, cp.b.RootfsPath, instList)

	if err = pacCmd.Run(); err != nil {
		return fmt.Errorf("while pacstrapping: %v", err)
	}

	//Pacman package signing setup
	cmd := exec.Command("arch-chroot", cp.b.RootfsPath, "/bin/sh", "-c", "haveged -w 1024; pacman-key --init; pacman-key --populate archlinux")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("while setting up package signing: %v", err)
	}

	//Clean up haveged
	cmd = exec.Command("arch-chroot", cp.b.RootfsPath, "pacman", "-Rs", "--noconfirm", "haveged")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("while cleaning up packages: %v", err)
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *ArchConveyorPacker) Pack(context.Context) (b *types.Bundle, err error) {
	err = cp.insertBaseEnv()
	if err != nil {
		return nil, fmt.Errorf("while inserting base environment: %v", err)
	}

	err = cp.insertRunScript()
	if err != nil {
		return nil, fmt.Errorf("while inserting runscript: %v", err)
	}

	return cp.b, nil
}

func (cp *ArchConveyorPacker) getPacConf(pacmanConfURL string) (pacConf string, err error) {
	pacConfFile, err := ioutil.TempFile(cp.b.RootfsPath, "pac-conf-")
	if err != nil {
		return
	}

	resp, err := http.Get(pacmanConfURL)
	if err != nil {
		return "", fmt.Errorf("while performing http request: %v", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(pacConfFile, resp.Body)
	if err != nil {
		return
	}

	return pacConfFile.Name(), nil
}

func (cp *ArchConveyorPacker) insertBaseEnv() (err error) {
	if err = makeBaseEnv(cp.b.RootfsPath); err != nil {
		return
	}
	return nil
}

func (cp *ArchConveyorPacker) insertRunScript() (err error) {
	err = ioutil.WriteFile(filepath.Join(cp.b.RootfsPath, "/.singularity.d/runscript"), []byte("#!/bin/sh\n"), 0755)
	if err != nil {
		return
	}

	return nil
}

// CleanUp removes any tmpfs owned by the conveyorPacker on the filesystem
func (cp *ArchConveyorPacker) CleanUp() {
	cp.b.Remove()
}

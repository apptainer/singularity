// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/priv"
	"github.com/sylabs/singularity/pkg/util/crypt"
)

/*
 * see https://github.com/opencontainers/runtime-spec/blob/master/runtime.md#lifecycle
 * we will run step 8/9 there
 */

// CleanupContainer cleans up the container
func (e *EngineOperations) CleanupContainer(fatal error, status syscall.WaitStatus) error {

	if e.EngineConfig.GetDeleteImage() {
		image := e.EngineConfig.GetImage()
		sylog.Verbosef("Removing image %s", image)
		sylog.Infof("Cleaning up image...")
		if err := os.RemoveAll(image); err != nil {
			sylog.Errorf("failed to delete container image %s: %s", image, err)
		}
	}

	if e.EngineConfig.Network != nil {
		if e.EngineConfig.GetFakeroot() {
			priv.Escalate()
		}
		if err := e.EngineConfig.Network.DelNetworks(); err != nil {
			sylog.Errorf("%s", err)
		}
		if e.EngineConfig.GetFakeroot() {
			priv.Drop()
		}
	}

	if e.EngineConfig.Cgroups != nil {
		if err := e.EngineConfig.Cgroups.Remove(); err != nil {
			sylog.Errorf("%s", err)
		}
	}

	if e.EngineConfig.GetInstance() {
		file, err := instance.Get(e.CommonConfig.ContainerID, instance.SingSubDir)
		if err != nil {
			return err
		}
		return file.Delete()
	}

	if e.EngineConfig.CryptDev != "" {
		cleanupCrypt(e.EngineConfig.CryptDev)
	}

	return nil
}

func cleanupCrypt(path string) error {

	// Elevate the privilege to unmount and delete the crypt device
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	uid := os.Getuid()
	err := syscall.Setresuid(uid, 0, uid)
	if err != nil {
		return fmt.Errorf("error while setting root uid")
	}

	defer syscall.Setresuid(uid, uid, 0)

	err = syscall.Unmount(filepath.Join(buildcfg.SESSIONDIR, "final"), syscall.MNT_DETACH)
	if err != nil {
		return fmt.Errorf("failed while unmounting final session directory: %s", err)
	}

	err = syscall.Unmount(filepath.Join(buildcfg.SESSIONDIR, "rootfs"), syscall.MNT_DETACH)
	if err != nil {
		return fmt.Errorf("error while unmounting rootfs session directory: %s", err)
	}

	devName := filepath.Base(path)

	cryptDev := &crypt.Device{}
	err = cryptDev.CloseCryptDevice(devName)
	if err != nil {
		return fmt.Errorf("unable to delete crypt device: %s", devName)
	}

	return nil
}

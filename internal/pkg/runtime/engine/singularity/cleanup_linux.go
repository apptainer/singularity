// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/instance"
	fakerootConfig "github.com/sylabs/singularity/internal/pkg/runtime/engine/fakeroot/config"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/priv"
	"github.com/sylabs/singularity/internal/pkg/util/starter"
	"github.com/sylabs/singularity/pkg/runtime/engine/config"
	"github.com/sylabs/singularity/pkg/util/crypt"
)

// CleanupContainer is called from master after the MonitorContainer returns.
// It is responsible for ensuring that the container has been properly torn down.
//
// Additional privileges may be gained when running
// in suid flow. However, when a user namespace is requested and it is not
// a hybrid workflow (e.g. fakeroot), then there is no privileged saved uid
// and thus no additional privileges can be gained.
//
// For better understanding of runtime flow in general refer to
// https://github.com/opencontainers/runtime-spec/blob/master/runtime.md#lifecycle.
// CleanupContainer is performing step 8/9 here.
func (e *EngineOperations) CleanupContainer(ctx context.Context, fatal error, status syscall.WaitStatus) error {
	if e.EngineConfig.GetDeleteImage() {
		image := e.EngineConfig.GetImage()
		sylog.Verbosef("Removing image %s", image)
		sylog.Infof("Cleaning up image...")

		var err error

		if e.EngineConfig.GetFakeroot() && os.Getuid() != 0 {
			// this is required when we are using SUID workflow
			// because master process is not in the fakeroot
			// context and can get permission denied error during
			// image removal, so we execute "rm -rf /tmp/image" via
			// the fakeroot engine
			err = fakerootCleanup(image)
		} else {
			err = os.RemoveAll(image)
		}
		if err != nil {
			sylog.Errorf("failed to delete container image %s: %s", image, err)
		}
	}

	if e.EngineConfig.Network != nil {
		if e.EngineConfig.GetFakeroot() {
			priv.Escalate()
		}
		if err := e.EngineConfig.Network.DelNetworks(ctx); err != nil {
			sylog.Errorf("could not delete networks: %v", err)
		}
		if e.EngineConfig.GetFakeroot() {
			priv.Drop()
		}
	}

	if e.EngineConfig.Cgroups != nil {
		if err := e.EngineConfig.Cgroups.Remove(); err != nil {
			sylog.Errorf("could not remove cgroups: %v", err)
		}
	}

	if e.EngineConfig.CryptDev != "" {
		if err := cleanupCrypt(e.EngineConfig.CryptDev); err != nil {
			sylog.Errorf("could not cleanup crypt: %v", err)
		}
	}

	if e.EngineConfig.GetInstance() {
		file, err := instance.Get(e.CommonConfig.ContainerID, instance.SingSubDir)
		if err != nil {
			return err
		}
		return file.Delete()
	}

	return nil
}

func cleanupCrypt(path string) error {
	// elevate the privilege to unmount and delete the crypt device
	priv.Escalate()
	defer priv.Drop()

	err := syscall.Unmount(filepath.Join(buildcfg.SESSIONDIR, "final"), syscall.MNT_DETACH)
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

func fakerootCleanup(path string) error {
	command := []string{"/bin/rm", "-rf", path}

	sylog.Debugf("Calling fakeroot engine to execute %q", strings.Join(command, " "))

	cfg := &config.Common{
		EngineName:   fakerootConfig.Name,
		ContainerID:  "fakeroot",
		EngineConfig: &fakerootConfig.EngineConfig{Args: command},
	}

	return starter.Run(
		"Singularity fakeroot",
		cfg,
		starter.UseSuid(true),
	)
}

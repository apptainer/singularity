// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/util/mainthread"

	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

/*
 * see https://github.com/opencontainers/runtime-spec/blob/master/runtime.md#lifecycle
 * we will run step 8/9 there
 */

// CleanupContainer cleans up the container
func (engine *EngineOperations) CleanupContainer(fatal error, status syscall.WaitStatus) error {
	sylog.Debugf("Cleanup container")

	if engine.EngineConfig.GetDeleteImage() {
		image := engine.EngineConfig.GetImage()
		sylog.Verbosef("Removing image %s", image)
		sylog.Infof("Cleaning up image...")
		if err := os.RemoveAll(image); err != nil {
			sylog.Errorf("failed to delete container image %s: %s", image, err)
		}
	}

	if engine.EngineConfig.Network != nil {
		if err := engine.EngineConfig.Network.DelNetworks(); err != nil {
			sylog.Errorf("%s", err)
		}
	}

	if engine.EngineConfig.Cgroups != nil {
		if err := engine.EngineConfig.Cgroups.Remove(); err != nil {
			sylog.Errorf("%s", err)
		}
	}

	if engine.EngineConfig.GetInstance() {
		uid := os.Getuid()

		file, err := instance.Get(engine.CommonConfig.ContainerID)
		if err != nil {
			return err
		}

		if file.PPid != os.Getpid() {
			return nil
		}

		if file.Privileged {
			var err error

			mainthread.Execute(func() {
				if err = syscall.Setresuid(0, 0, uid); err != nil {
					err = fmt.Errorf("failed to escalate privileges")
					return
				}
				defer syscall.Setresuid(uid, uid, 0)

				if err = file.Delete(); err != nil {
					return
				}
			})
			return err
		}
		return file.Delete()
	}

	return nil
}

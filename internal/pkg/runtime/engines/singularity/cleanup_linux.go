// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"os"
	"syscall"

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
		file, err := instance.Get(engine.CommonConfig.ContainerID, instance.SingSubDir)
		if err != nil {
			return err
		}
		return file.Delete()
	}

	return nil
}

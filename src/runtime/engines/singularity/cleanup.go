// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/util/mainthread"

	"github.com/singularityware/singularity/src/pkg/instance"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

/*
 * see https://github.com/opencontainers/runtime-spec/blob/master/runtime.md#lifecycle
 * we will run step 8/9 there
 */

// CleanupContainer cleans up the container
func (engine *EngineOperations) CleanupContainer() error {
	sylog.Debugf("Cleanup container")

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
				if err := syscall.Setresuid(0, 0, uid); err != nil {
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

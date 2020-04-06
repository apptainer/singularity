// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"context"
	"fmt"

	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/util/exec"
	"github.com/sylabs/singularity/pkg/ociruntime"
	"github.com/sylabs/singularity/pkg/sylog"
)

// OciDelete deletes container resources
func OciDelete(ctx context.Context, containerID string) error {
	engineConfig, err := getEngineConfig(containerID)
	if err != nil {
		return err
	}

	switch engineConfig.State.Status {
	case ociruntime.Running:
		return fmt.Errorf("cannot delete '%s', the state of the container must be created or stopped", containerID)
	case ociruntime.Stopped:
	case ociruntime.Created:
		if err := OciKill(containerID, "SIGTERM", 2); err != nil {
			return err
		}
		engineConfig, err = getEngineConfig(containerID)
		if err != nil {
			return err
		}
	}

	hooks := engineConfig.OciConfig.Hooks
	if hooks != nil {
		for _, h := range hooks.Poststop {
			if err := exec.Hook(ctx, &h, &engineConfig.State.State); err != nil {
				sylog.Warningf("%s", err)
			}
		}
	}

	// remove instance files
	file, err := instance.Get(containerID, instance.OciSubDir)
	if err != nil {
		return err
	}
	return file.Delete()
}

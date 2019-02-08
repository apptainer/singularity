// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"

	"github.com/sylabs/singularity/internal/pkg/cgroups"
	"github.com/sylabs/singularity/pkg/ociruntime"
)

// OciPauseResume pauses/resumes processes in a container
func OciPauseResume(containerID string, pause bool) error {
	state, err := getState(containerID)
	if err != nil {
		return err
	}

	if state.Status != ociruntime.Running {
		return fmt.Errorf("container %s is not running", containerID)
	}

	manager := &cgroups.Manager{Pid: state.State.Pid}

	if !pause {
		return manager.Resume()
	}

	return manager.Pause()
}

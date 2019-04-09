// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/sylabs/singularity/pkg/ociruntime"
	"github.com/sylabs/singularity/pkg/util/unix"
)

// OciPauseResume pauses/resumes processes in a container
func OciPauseResume(containerID string, pause bool) error {
	state, err := getState(containerID)
	if err != nil {
		return err
	}

	if state.ControlSocket == "" {
		return fmt.Errorf("can't find control socket")
	}

	if pause && state.Status != ociruntime.Running {
		return fmt.Errorf("container %s is not running", containerID)
	} else if !pause && state.Status != ociruntime.Paused {
		return fmt.Errorf("container %s is not paused", containerID)
	}

	ctrl := &ociruntime.Control{}
	if pause {
		ctrl.Pause = true
	} else {
		ctrl.Resume = true
	}

	c, err := unix.Dial(state.ControlSocket)
	if err != nil {
		return fmt.Errorf("failed to connect to control socket")
	}
	defer c.Close()

	enc := json.NewEncoder(c)
	if enc == nil {
		return fmt.Errorf("cannot instantiate new JSON encoder")
	}

	if err := enc.Encode(ctrl); err != nil {
		return err
	}

	// wait runtime close socket connection for ACK
	d := make([]byte, 1)
	if _, err := c.Read(d); err != io.EOF {
		return err
	}

	// check status
	state, err = getState(containerID)
	if err != nil {
		return err
	}
	if pause && state.Status != ociruntime.Paused {
		return fmt.Errorf("bad status %s returned instead of paused", state.Status)
	} else if !pause && state.Status != ociruntime.Running {
		return fmt.Errorf("bad status %s returned instead of running", state.Status)
	}

	return nil
}

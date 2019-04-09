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

// OciStart starts a previously create container
func OciStart(containerID string) error {
	state, err := getState(containerID)
	if err != nil {
		return err
	}

	if state.Status != ociruntime.Created {
		return fmt.Errorf("cannot start '%s', the state of the container must be %s", containerID, ociruntime.Created)
	}

	if state.ControlSocket == "" {
		return fmt.Errorf("can't find control socket")
	}

	ctrl := &ociruntime.Control{}
	ctrl.StartContainer = true

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

	return nil
}

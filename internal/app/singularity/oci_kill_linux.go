// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"io"
	"syscall"
	"time"

	"github.com/sylabs/singularity/internal/pkg/util/signal"
	"github.com/sylabs/singularity/pkg/ociruntime"
	"github.com/sylabs/singularity/pkg/util/unix"
)

// OciKill kills container process
func OciKill(containerID string, killSignal string, killTimeout int) error {
	// send signal to the instance
	state, err := getState(containerID)
	if err != nil {
		return err
	}

	if state.Status != ociruntime.Created && state.Status != ociruntime.Running {
		return fmt.Errorf("cannot kill '%s', the state of the container must be created or running", containerID)
	}

	sig := syscall.SIGTERM

	if killSignal != "" {
		sig, err = signal.Convert(killSignal)
		if err != nil {
			return err
		}
	}

	if killTimeout > 0 {
		c, err := unix.Dial(state.ControlSocket)
		if err != nil {
			return fmt.Errorf("failed to connect to control socket")
		}
		defer c.Close()

		killed := make(chan bool, 1)

		go func() {
			// wait runtime close socket connection for ACK
			d := make([]byte, 1)
			if _, err := c.Read(d); err == io.EOF {
				killed <- true
			}
		}()

		if err := syscall.Kill(state.Pid, sig); err != nil {
			return err
		}

		select {
		case <-killed:
		case <-time.After(time.Duration(killTimeout) * time.Second):
			return syscall.Kill(state.Pid, syscall.SIGKILL)
		}
	} else {
		return syscall.Kill(state.Pid, sig)
	}

	return nil
}

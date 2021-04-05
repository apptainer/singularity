// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/pkg/ociruntime"
	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/sylabs/singularity/pkg/util/unix"
)

// OciRun runs a container (equivalent to create/start/delete)
func OciRun(ctx context.Context, containerID string, args *OciArgs) error {
	dir, err := instance.GetDir(containerID, instance.OciSubDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	args.SyncSocketPath = filepath.Join(dir, "run.sock")

	l, err := unix.CreateSocket(args.SyncSocketPath)
	if err != nil {
		os.Remove(args.SyncSocketPath)
		return err
	}

	defer l.Close()

	status := make(chan string, 1)

	if err := OciCreate(containerID, args); err != nil {
		defer os.Remove(args.SyncSocketPath)
		if _, err1 := getState(containerID); err1 != nil {
			return err
		}
		if err := OciDelete(ctx, containerID); err != nil {
			sylog.Warningf("can't delete container %s", containerID)
		}
		return err
	}

	defer exitContainer(ctx, containerID, true)
	defer os.Remove(args.SyncSocketPath)

	go func() {
		var state specs.State

		for {
			c, err := l.Accept()
			if err != nil {
				status <- err.Error()
				return
			}

			dec := json.NewDecoder(c)
			if err := dec.Decode(&state); err != nil {
				status <- err.Error()
				return
			}

			c.Close()

			switch state.Status {
			case ociruntime.Created:
				// ignore error there and wait for stopped status
				OciStart(containerID)
			case ociruntime.Running:
				status <- string(state.Status)
			case ociruntime.Stopped:
				status <- string(state.Status)
			}
		}
	}()

	// wait running status
	s := <-status
	if s != ociruntime.Running {
		return fmt.Errorf("%s", s)
	}

	engineConfig, err := getEngineConfig(containerID)
	if err != nil {
		return err
	}

	if err := attach(engineConfig, true); err != nil {
		// kill container before deletion
		sylog.Errorf("%s", err)
		OciKill(containerID, "SIGKILL", 1)
		return err
	}

	// wait stopped status
	s = <-status
	if s != ociruntime.Stopped {
		return fmt.Errorf("%s", s)
	}

	return nil
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"fmt"
	"os"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/ociruntime"
)

// CleanupContainer cleans up the container
func (e *EngineOperations) CleanupContainer(fatal error, status syscall.WaitStatus) error {
	if e.EngineConfig.Cgroups != nil {
		e.EngineConfig.Cgroups.Remove()
	}

	pidFile := e.EngineConfig.GetPidFile()
	if pidFile != "" {
		os.Remove(pidFile)
	}

	// if container wasn't created, delete instance files
	if e.EngineConfig.State.Status == ociruntime.Creating {
		name := e.CommonConfig.ContainerID
		file, err := instance.Get(name, instance.OciSubDir)
		if err != nil {
			sylog.Warningf("no instance files found for %s: %s", name, err)
			return nil
		}
		if err := file.Delete(); err != nil {
			sylog.Warningf("failed to delete instance files: %s", err)
		}
		return nil
	}

	exitCode := 0
	desc := ""

	if fatal != nil {
		exitCode = 255
		desc = fatal.Error()
	} else if status.Signaled() {
		s := status.Signal()
		exitCode = int(s) + 128
		desc = fmt.Sprintf("interrupted by signal %s", s.String())
	} else {
		exitCode = status.ExitStatus()
		desc = fmt.Sprintf("exited with code %d", status.ExitStatus())
	}

	e.EngineConfig.State.ExitCode = &exitCode
	e.EngineConfig.State.ExitDesc = desc

	if err := e.updateState(ociruntime.Stopped); err != nil {
		return err
	}

	if e.EngineConfig.State.AttachSocket != "" {
		os.Remove(e.EngineConfig.State.AttachSocket)
	}
	if e.EngineConfig.State.ControlSocket != "" {
		os.Remove(e.EngineConfig.State.ControlSocket)
	}

	return nil
}

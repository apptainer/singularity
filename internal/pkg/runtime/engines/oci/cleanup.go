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
	"github.com/sylabs/singularity/internal/pkg/util/exec"
)

// CleanupContainer cleans up the container
func (engine *EngineOperations) CleanupContainer(fatal error, status syscall.WaitStatus) error {
	if engine.EngineConfig.Cgroups != nil {
		engine.EngineConfig.Cgroups.Remove()
	}

	pidFile := engine.EngineConfig.GetPidFile()
	if pidFile != "" {
		os.Remove(pidFile)
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

	engine.EngineConfig.State.ExitCode = &exitCode
	engine.EngineConfig.State.ExitDesc = desc

	if engine.EngineConfig.State.Status != "running" {
		hooks := engine.EngineConfig.OciConfig.Hooks
		if hooks != nil {
			for _, h := range hooks.Poststop {
				if err := exec.Hook(&h, &engine.EngineConfig.State.State); err != nil {
					sylog.Warningf("%s", err)
				}
			}
		}
		// remove instance files
		file, err := instance.Get(engine.CommonConfig.ContainerID)
		if err != nil {
			return err
		}
		return file.Delete()
	}

	if err := engine.updateState("stopped"); err != nil {
		return err
	}

	if engine.EngineConfig.State.AttachSocket != "" {
		os.Remove(engine.EngineConfig.State.AttachSocket)
	}
	if engine.EngineConfig.State.ControlSocket != "" {
		os.Remove(engine.EngineConfig.State.ControlSocket)
	}

	return nil
}

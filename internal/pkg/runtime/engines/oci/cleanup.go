// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/exec"
	"github.com/sylabs/singularity/pkg/ociruntime"
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

	exitCode := "0"
	desc := ""

	if fatal != nil {
		exitCode = strconv.FormatInt(int64(255), 10)
		desc = fatal.Error()
	} else if status.Signaled() {
		s := status.Signal()
		exitCode = fmt.Sprintf("%d", s+128)
		desc = fmt.Sprintf("interrupted by signal %s", s.String())
	} else {
		exitCode = strconv.FormatInt(int64(status.ExitStatus()), 10)
		desc = fmt.Sprintf("exited with code %d", status.ExitStatus())
	}

	engine.EngineConfig.State.Annotations[ociruntime.AnnotationExitCode] = exitCode
	engine.EngineConfig.State.Annotations[ociruntime.AnnotationExitDesc] = desc

	if engine.EngineConfig.State.Status != "running" {
		hooks := engine.EngineConfig.OciConfig.Hooks
		if hooks != nil {
			for _, h := range hooks.Poststop {
				if err := exec.Hook(&h, &engine.EngineConfig.State); err != nil {
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

	os.Remove(engine.EngineConfig.State.Annotations[ociruntime.AnnotationAttachSocket])
	os.Remove(engine.EngineConfig.State.Annotations[ociruntime.AnnotationControlSocket])

	return nil
}

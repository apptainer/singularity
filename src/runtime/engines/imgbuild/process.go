// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"os"
	"os/exec"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

// PrestartProcess _
func (e *EngineOperations) PrestartProcess() error {
	return nil
}

// StartProcess runs the %post script
func (e *EngineOperations) StartProcess() error {
	// Run %post script here

	post := exec.Command("/bin/sh", "-c", e.EngineConfig.Def.BuildData.Post)

	_, err := post.Output()

	if err != nil {
		sylog.Errorf("Error running script: %v", err)
		os.Exit(1)
	}

	os.Exit(0)
	return nil
}

// MonitorContainer _
func (e *EngineOperations) MonitorContainer() error {
	return nil
}

// CleanupContainer _
func (e *EngineOperations) CleanupContainer() error {
	return nil
}

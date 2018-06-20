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

	post := exec.Command("/bin/sh", "-c", e.EngineConfig.Recipe.BuildData.Post)
	post.Stdout = os.Stdout
	post.Stderr = os.Stderr

	sylog.Infof("Running %%post script\n")
	if err := post.Start(); err != nil {
		sylog.Fatalf("failed to start post proc: %v\n", err)
	}
	if err := post.Wait(); err != nil {
		sylog.Fatalf("post proc: %v\n", err)
	}
	sylog.Infof("Finished running %%post script. exit status 0\n")

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

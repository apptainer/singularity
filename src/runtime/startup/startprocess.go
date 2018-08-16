// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
)

// StartProcess is the last stage. This is run as the child process of SMaster stage,
// allowing SMaster to have the ability to monitor or trace the contained process
func StartProcess(masterSocket int, startupConfig *config.Startup, engineConfig []byte) {
	engine, err := engines.NewEngine(engineConfig)
	if err != nil {
		sylog.Fatalf("Failed to create container engine [engines.NewEngine()]: %s\n", err)
	}

	conn := getConnFromSocket(masterSocket, "master-socket")
	if err := engine.StartProcess(conn); err != nil {
		sylog.Fatalf("Failed to start container process: %s\n", err)
	}

	sylog.Fatalf("This should be completely impossible to reach, please leave\n")
}

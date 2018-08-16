// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"os"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
)

// PrepareStartup is the first stage of the container execution process. This stage has
// _two_ primary functions:
//
//     * Validate the Engine Configuration JSON which was passed in
//     * Populate the Startup Configuration object for the startup binary to consume
//
// PrepareStartup will then pass the validated Engine AND Startup Configurations back to the startup binary
// by writing the raw data to os.Stdout
func PrepareStartup(masterSocket int, startupConfig *config.Startup, engineConfig []byte) {
	engine, err := engines.NewEngine(engineConfig)
	if err != nil {
		sylog.Fatalf("Failed to create container engine [engines.NewEngine()]: %s\n", err)
	}

	conn := getConnFromSocket(masterSocket, "master-socket")
	if err := engine.PrepareEngineConfig(conn); err != nil {
		sylog.Fatalf("Failed to prepare EngineConfig [PrepareEngineConfig()]: %s\n", err)
	}

	if err := engine.PrepareStartupConfig(startupConfig); err != nil {
		sylog.Fatalf("Failed to prepare StartupConfig [PrepareStartupConfig()]: %s\n", err)
	}

	if err := startupConfig.WritePayload(os.Stdout, engine.Common); err != nil {
		sylog.Fatalf("%s", err)
	}
	os.Exit(0)
}

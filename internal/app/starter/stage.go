// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package starter

import (
	"net"
	"os"

	"github.com/sylabs/singularity/internal/pkg/runtime/engines"
	sarterConfig "github.com/sylabs/singularity/internal/pkg/runtime/engines/config/starter"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// Stage performs container startup.
func Stage(stage, masterSocket int, sconfig *sarterConfig.Config, jsonConfig []byte) {
	var conn net.Conn
	var err error

	if masterSocket != -1 {
		comm := os.NewFile(uintptr(masterSocket), "master-socket")
		conn, err = net.FileConn(comm)
		if err != nil {
			sylog.Fatalf("failed to copy master unix socket descriptor: %s", err)
			return
		}
		comm.Close()
	} else {
		conn = nil
	}

	engine, err := engines.NewEngine(jsonConfig)
	if err != nil {
		sylog.Fatalf("failed to initialize runtime engine: %s\n", err)
	}

	if stage == 1 {
		sylog.Debugf("Entering scontainer stage 1\n")
		if err := engine.PrepareConfig(conn, sconfig); err != nil {
			sylog.Fatalf("%s\n", err)
		}
		if err := sconfig.Write(engine.Common); err != nil {
			sylog.Fatalf("%s", err)
		}
		os.Exit(0)
	} else {
		if err := engine.StartProcess(conn); err != nil {
			sylog.Fatalf("%s\n", err)
		}
	}
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"net"
	"os"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines"
	"github.com/singularityware/singularity/src/runtime/engines/config/starter"
)

// SContainer performs container startup
func SContainer(stage int, masterSocket int, starterConfig *starter.Config, jsonBytes []byte) {
	var conn net.Conn
	var err error

	if masterSocket != -1 {
		comm := os.NewFile(uintptr(masterSocket), "master-socket")
		conn, err = net.FileConn(comm)
		if err != nil {
			sylog.Fatalf("failed to copy master unix socket descriptor: %s", err)
			return
		}
		if stage == 2 {
			comm.Close()
		}
	} else {
		conn = nil
	}

	engine, err := engines.NewEngine(jsonBytes)
	if err != nil {
		sylog.Fatalf("failed to initialize runtime engine: %s\n", err)
	}

	if stage == 1 {
		sylog.Debugf("Entering scontainer stage 1\n")

		if err := engine.PrepareConfig(conn, starterConfig); err != nil {
			sylog.Fatalf("%s\n", err)
		}

		if err := starterConfig.WritePayload(conn, engine.Common); err != nil {
			sylog.Fatalf("%s", err)
		}
		conn.Close()
		os.Exit(0)
	} else {
		if err := engine.StartProcess(conn); err != nil {
			sylog.Fatalf("%s\n", err)
		}
	}
}

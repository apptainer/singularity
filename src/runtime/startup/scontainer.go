// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"net"
	"os"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines"
	"github.com/singularityware/singularity/src/runtime/engines/common/config/wrapper"
)

// SContainer performs container startup
func SContainer(stage int, masterSocket int, wrapperConfig *wrapper.Config, jsonBytes []byte) {
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

	engine, err := engines.NewEngine(jsonBytes)
	if err != nil {
		sylog.Fatalf("failed to initialize runtime engine: %s\n", err)
	}

	if stage == 1 {
		sylog.Debugf("Entering scontainer stage 1\n")

		if err := engine.PrepareConfig(conn, wrapperConfig); err != nil {
			sylog.Fatalf("%s\n", err)
		}

		wrapperConfig.WritePayload(os.Stdout, engine.Common)
		os.Exit(0)
	} else {
		if err := engine.StartProcess(conn); err != nil {
			sylog.Fatalf("%s\n", err)
		}
	}
}

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
)

// RPCServer serves runtime engine requests
func RPCServer(socket int, runtime string) {
	comm := os.NewFile(uintptr(socket), "unix")

	conn, err := net.FileConn(comm)
	if err != nil {
		sylog.Fatalf("socket communication error: %s\n", err)
	}
	comm.Close()

	engines.ServeRuntimeEngineRequests(runtime, conn)
}

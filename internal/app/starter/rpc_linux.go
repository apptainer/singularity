// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package starter

import (
	"net"
	"os"

	"github.com/sylabs/singularity/internal/pkg/runtime/engine"
	"github.com/sylabs/singularity/pkg/sylog"
)

// RPCServer serves runtime engine requests.
//
// The RPC server process is already in correct namespaces
// required by container, so any operations performed will
// affect final container environment. When run with suid
// flow, i.e. no user namespace for container is created
// and no hybrid workflow is requested, the server is run
// with escalated privileges (as euid 0).
func RPCServer(socket int, e *engine.Engine) {
	comm := os.NewFile(uintptr(socket), "unix")
	conn, err := net.FileConn(comm)
	if err != nil {
		sylog.Fatalf("socket communication error: %s\n", err)
	}
	comm.Close()
	engine.ServeRPCRequests(e, conn)

	os.Exit(0)
}

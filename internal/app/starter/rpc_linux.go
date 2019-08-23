// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package starter

import (
	"net"
	"os"

	"github.com/sylabs/singularity/internal/pkg/runtime/engines/engine"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// RPCServer serves runtime engine requests.
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

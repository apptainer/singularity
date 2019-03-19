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
func Stage(stage, masterSocket int, sconfig *sarterConfig.Config, engine *engines.Engine) {
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

	if stage == 1 {
		sylog.Debugf("Entering stage 1\n")
		if err := engine.PrepareConfig(sconfig); err != nil {
			sylog.Fatalf("%s\n", err)
		}
		if err := sconfig.Write(engine.Common); err != nil {
			sylog.Fatalf("%s", err)
		}
		os.Exit(0)
	} else {
		sylog.Debugf("Entering stage 2\n")
		if err := engine.StartProcess(conn); err != nil {
			// write data to just tell master to not execute PostStartProcess
			// in case of failure
			if _, err := conn.Write([]byte("f")); err != nil {
				sylog.Errorf("fail to send data to master: %s", err)
			}
			sylog.Fatalf("%s\n", err)
		}
	}
}

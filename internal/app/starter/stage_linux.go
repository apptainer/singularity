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

// StageOne performs container configuration.
func StageOne(sconfig *sarterConfig.Config, engine *engines.Engine) {
	sylog.Debugf("Entering stage 1\n")

	// call engine operation PrepareConfig, at this stage
	// we are running without any privileges
	if err := engine.PrepareConfig(sconfig); err != nil {
		sylog.Fatalf("%s\n", err)
	}

	// store (possibly) updated engine configuration in
	// shared memory in order to pass it to stage 2 and
	// master processes
	if err := sconfig.Write(engine.Common); err != nil {
		sylog.Fatalf("%s", err)
	}

	os.Exit(0)
}

// StageTwo performs container execution.
func StageTwo(masterSocket int, engine *engines.Engine) {
	sylog.Debugf("Entering stage 2\n")

	// master socket allows communications between
	// stage 2 and master process, typically used for
	// synchronization or for sending state
	comm := os.NewFile(uintptr(masterSocket), "master-socket")
	conn, err := net.FileConn(comm)
	comm.Close()
	if err != nil {
		sylog.Fatalf("failed to copy master unix socket descriptor: %s", err)
		return
	}

	// call engine operation StartProcess, at this stage
	// we are in a container context, chroot was done.
	// The privileges are those applied by the container
	// configuration, in the case of Singularity engine
	// and if run as a user, there is no privileges set
	if err := engine.StartProcess(conn); err != nil {
		// write data to just tell master to not execute PostStartProcess
		// in case of failure
		if _, err := conn.Write([]byte("f")); err != nil {
			sylog.Errorf("fail to send data to master: %s", err)
		}
		sylog.Fatalf("%s\n", err)
	}
}

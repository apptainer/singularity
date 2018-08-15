// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

/*
#include "c/startup.c"
*/
// #cgo CFLAGS: -I..
import "C"

import (
	"net"
	"os"
	"unsafe"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
)

func main() {
	loglevel := os.Getenv("SINGULARITY_MESSAGELEVEL")

	os.Clearenv()

	if loglevel != "" {
		if os.Setenv("SINGULARITY_MESSAGELEVEL", loglevel) != nil {
			sylog.Warningf("can't restore SINGULARITY_MESSAGELEVEL environment variable")
		}
	}

	cconf := unsafe.Pointer(&C.config)
	startupConfig := config.NewStartupConfig(config.CStartupConfig(cconf))
	engineConfig := C.GoBytes(unsafe.Pointer(C.json_stdin), C.int(startupConfig.GetJSONConfSize()))

	switch C.execute {
	case C.STAGE_PREPARESTARTUP: // PrepareEngineConfing() => PrepareStartupConfig()
		sylog.Debugf("Running PrepareStartup stage [PrepareEngineConfig() + PrepareStartupConfig()]\n")
		PrepareStartup(int(C.master_socket[1]), startupConfig, engineConfig)
	case C.STAGE_JOINCONTAINER: // StartProcess()
		sylog.Debugf("Runninig JoinContainer stage [StartProcess()]\n")
		StartProcess(int(C.master_socket[1]), startupConfig, engineConfig)
	case C.STAGE_SMASTER: // CreateContainer() => MonitorContainer() => CleanupContainer()
		sylog.Debugf("Running SMaster stage [CreateContainer() => MonitorContainer() => CleanupContainer()]\n")
		SMaster(int(C.rpc_socket[0]), int(C.master_socket[0]), startupConfig, engineConfig)
	case C.STAGE_STARTCONTAINER:
		sylog.Debugf("Running NewContainer stage [RPCServer + StartProcess()]\n")
		RPCServer(int(C.rpc_socket[1]), C.GoString(C.sruntime))

		C.prepare_scontainer_stage(C.STAGE_JOINCONTAINER)
		StartProcess(int(C.master_socket[1]), startupConfig, engineConfig)
	}
	sylog.Fatalf("You should not be there\n")
}

func getConnFromSocket(fd int, name string) (conn net.Conn) {
	var err error

	if fd != -1 {
		comm := os.NewFile(uintptr(fd), name)
		conn, err = net.FileConn(comm)
		if err != nil {
			sylog.Fatalf("Failed to copy %s file descriptor: %s", name, err)
		}
		comm.Close()
	} else {
		conn = nil
	}

	return conn
}

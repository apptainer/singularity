// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

/*
#include "c/starter.c"
*/
// #cgo CFLAGS: -I..
import "C"

import (
	"os"
	"runtime"
	"unsafe"

	"github.com/sylabs/singularity/internal/app/starter"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines"
	starterConfig "github.com/sylabs/singularity/internal/pkg/runtime/engines/config/starter"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/mainthread"
)

func startup() {
	loglevel := os.Getenv("SINGULARITY_MESSAGELEVEL")
	os.Clearenv()
	if loglevel != "" {
		if os.Setenv("SINGULARITY_MESSAGELEVEL", loglevel) != nil {
			sylog.Warningf("can't restore SINGULARITY_MESSAGELEVEL environment variable")
		}
	}

	cconf := unsafe.Pointer(&C.config)
	sconfig := starterConfig.NewConfig(starterConfig.CConfig(cconf))
	jsonBytes := C.GoBytes(unsafe.Pointer(C.json_stdin), C.int(sconfig.GetJSONConfSize()))

	// free allocated buffer
	C.free(unsafe.Pointer(C.json_stdin))
	if unsafe.Pointer(C.nspath) != nil {
		C.free(unsafe.Pointer(C.nspath))
	}

	switch C.execute {
	case C.SCONTAINER_STAGE1:
		sylog.Verbosef("Execute scontainer stage 1\n")
		starter.Stage(int(C.SCONTAINER_STAGE1), int(C.master_socket[1]), sconfig, jsonBytes)
	case C.SCONTAINER_STAGE2:
		sylog.Verbosef("Execute scontainer stage 2\n")
		mainthread.Execute(func() {
			starter.Stage(int(C.SCONTAINER_STAGE2), int(C.master_socket[1]), sconfig, jsonBytes)
		})
	case C.SMASTER:
		sylog.Verbosef("Execute smaster process\n")
		starter.Master(int(C.rpc_socket[0]), int(C.master_socket[0]), sconfig, jsonBytes)
	case C.RPC_SERVER:
		sylog.Verbosef("Serve RPC requests\n")
		starter.RPCServer(int(C.rpc_socket[1]), C.GoString(C.sruntime))
	}
	sylog.Fatalf("You should not be there\n")
}

func init() {
	// lock main thread for function execution loop
	runtime.LockOSThread()
	// this is mainly to reduce memory footprint
	runtime.GOMAXPROCS(1)
}

func main() {
	// initialize runtime engines
	engines.Init()

	go startup()

	// run functions requiring execution in main thread
	for f := range mainthread.FuncChannel {
		f()
	}
}

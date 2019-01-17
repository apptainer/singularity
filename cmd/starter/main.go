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
	"github.com/sylabs/singularity/internal/pkg/util/goversion"
	"github.com/sylabs/singularity/internal/pkg/util/mainthread"
)

func getEngine(jsonConfig []byte) *engines.Engine {
	engine, err := engines.NewEngine(jsonConfig)
	if err != nil {
		sylog.Fatalf("failed to initialize runtime: %s\n", err)
	}
	return engine
}

func startup() {
	loglevel := os.Getenv("SINGULARITY_MESSAGELEVEL")
	os.Clearenv()
	if loglevel != "" {
		if os.Setenv("SINGULARITY_MESSAGELEVEL", loglevel) != nil {
			sylog.Warningf("can't restore SINGULARITY_MESSAGELEVEL environment variable")
		}
	}

	cconf := unsafe.Pointer(C.config)
	sconfig := starterConfig.NewConfig(starterConfig.CConfig(cconf))
	jsonConfig := sconfig.GetJSONConfig()

	switch C.execute {
	case C.STAGE1:
		sylog.Verbosef("Execute stage 1\n")
		starter.Stage(int(C.STAGE1), int(C.master_socket[1]), sconfig, getEngine(jsonConfig))
	case C.STAGE2:
		sylog.Verbosef("Execute stage 2\n")
		if err := sconfig.Release(); err != nil {
			sylog.Fatalf("%s", err)
		}

		mainthread.Execute(func() {
			starter.Stage(int(C.STAGE2), int(C.master_socket[1]), sconfig, getEngine(jsonConfig))
		})
	case C.MASTER:
		sylog.Verbosef("Execute master process\n")

		isInstance := sconfig.GetInstance()
		pid := sconfig.GetContainerPid()

		if err := sconfig.Release(); err != nil {
			sylog.Fatalf("%s", err)
		}

		starter.Master(int(C.rpc_socket[0]), int(C.master_socket[0]), isInstance, pid, getEngine(jsonConfig))
	case C.RPC_SERVER:
		sylog.Verbosef("Serve RPC requests\n")

		if err := sconfig.Release(); err != nil {
			sylog.Fatalf("%s", err)
		}

		name := engines.GetName(jsonConfig)
		starter.RPCServer(int(C.rpc_socket[1]), name)
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
	if err := goversion.Check(); err != nil {
		sylog.Fatalf("%s", err)
	}

	// initialize runtime engines
	engines.Init()

	go startup()

	// run functions requiring execution in main thread
	for f := range mainthread.FuncChannel {
		f()
	}
}

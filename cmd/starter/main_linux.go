// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

/*
#include "c/message.c"
#include "c/capability.c"
#include "c/setns.c"
#include "c/starter.c"
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/sylabs/singularity/internal/app/starter"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines"
	starterConfig "github.com/sylabs/singularity/internal/pkg/runtime/engines/config/starter"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	_ "github.com/sylabs/singularity/internal/pkg/util/goversion"
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
	sconfig := starterConfig.NewConfig(starterConfig.SConfig(unsafe.Pointer(C.sconfig)))
	jsonConfig := sconfig.GetJSONConfig()

	switch C.goexecute {
	case C.STAGE1:
		sylog.Verbosef("Execute stage 1\n")
		starter.StageOne(sconfig, getEngine(jsonConfig))
	case C.STAGE2:
		sylog.Verbosef("Execute stage 2\n")
		if err := sconfig.Release(); err != nil {
			sylog.Fatalf("%s", err)
		}

		mainthread.Execute(func() {
			starter.StageTwo(int(C.master_socket[1]), getEngine(jsonConfig))
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

// called after "starter.c" __init__ function returns.
func init() {
	// lock main thread for function execution loop
	runtime.LockOSThread()
	// this is mainly to reduce memory footprint
	runtime.GOMAXPROCS(1)
}

func main() {
	// initialize runtime engines
	engines.Init()

	// spawn a goroutine to use mainthread later
	go startup()

	// run functions requiring execution in main thread
	for f := range mainthread.FuncChannel {
		f()
	}
}

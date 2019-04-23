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
	"os"
	"runtime"
	"unsafe"

	"github.com/sylabs/singularity/internal/app/starter"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines"
	starterConfig "github.com/sylabs/singularity/internal/pkg/runtime/engines/config/starter"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/imgbuild"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/oci"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	_ "github.com/sylabs/singularity/internal/pkg/util/goversion"
	"github.com/sylabs/singularity/internal/pkg/util/mainthread"
)

func startup() {
	// global variable defined in cmd/starter/c/starter.c
	// C.config points to a shared memory area
	cconf := unsafe.Pointer(C.config)
	// initialize starter configuration
	sconfig := starterConfig.NewConfig(starterConfig.CConfig(cconf))
	// get JSON configuration originally passed from CLI
	jsonConfig := sconfig.GetJSONConfig()

	// get runtime engine name
	name := engines.GetName(jsonConfig)
	if name == "" {
		sylog.Fatalf("no runtime engine selected")
	}

	// initialize corresponding runtime engine
	if err := singularity.Init(name); err != nil {
		sylog.Fatalf("%s", err)
	}
	if err := oci.Init(name); err != nil {
		sylog.Fatalf("%s", err)
	}
	if err := imgbuild.Init(name); err != nil {
		sylog.Fatalf("%s", err)
	}

	// get engine operations previously registered
	// with corresponding engine's Init() functions
	engine, err := engines.NewEngine(jsonConfig)
	if err != nil {
		sylog.Fatalf("failed to initialize runtime engine: %s\n", err)
	}
	sylog.Debugf("%s runtime engine selected", engine.EngineName)

	switch C.execute {
	case C.STAGE1:
		sylog.Verbosef("Execute stage 1\n")
		starter.Stage(int(C.STAGE1), int(C.master_socket[1]), sconfig, engine)
	case C.STAGE2:
		sylog.Verbosef("Execute stage 2\n")
		if err := sconfig.Release(); err != nil {
			sylog.Fatalf("%s", err)
		}

		mainthread.Execute(func() {
			starter.Stage(int(C.STAGE2), int(C.master_socket[1]), sconfig, engine)
		})
	case C.MASTER:
		sylog.Verbosef("Execute master process\n")

		isInstance := sconfig.GetInstance()
		pid := sconfig.GetContainerPid()

		if err := sconfig.Release(); err != nil {
			sylog.Fatalf("%s", err)
		}

		starter.Master(int(C.rpc_socket[0]), int(C.master_socket[0]), isInstance, pid, engine)
	case C.RPC_SERVER:
		sylog.Verbosef("Serve RPC requests\n")

		if err := sconfig.Release(); err != nil {
			sylog.Fatalf("%s", err)
		}

		starter.RPCServer(int(C.rpc_socket[1]), name)
	}
	sylog.Fatalf("You should not be there\n")
}

func init() {
	// clear environment variable for Go context
	loglevel := os.Getenv("SINGULARITY_MESSAGELEVEL")
	os.Clearenv()
	if loglevel != "" {
		if os.Setenv("SINGULARITY_MESSAGELEVEL", loglevel) != nil {
			sylog.Warningf("can't restore SINGULARITY_MESSAGELEVEL environment variable")
		}
	}

	// lock main thread for function execution loop
	runtime.LockOSThread()
	// this is mainly to reduce memory footprint
	runtime.GOMAXPROCS(1)
}

func main() {
	go startup()

	// run functions requiring execution in main thread
	for f := range mainthread.FuncChannel {
		f()
	}
}

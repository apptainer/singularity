// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

/*
#include <sys/types.h>
#include "startup/c/wrapper.h"
*/
// #cgo CFLAGS: -I..
import "C"

import (
	"encoding/json"
	"os"
	"syscall"
	"unsafe"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines"
)

func bool2int(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

// SContainer performs container startup
//export SContainer
func SContainer(stage C.int, config *C.struct_cConfig, jsonC *C.char) {
	cconf := config

	/* get json configuration */
	sylog.Debugf("cconf.jsonConfSize: %d\n", C.int(cconf.jsonConfSize))
	jsonBytes := C.GoBytes(unsafe.Pointer(jsonC), C.int(cconf.jsonConfSize))

	engine, err := engines.NewEngine(jsonBytes)
	if err != nil {
		sylog.Fatalf("failed to initialize runtime engine: %s\n", err)
	}

	if stage == 1 {
		sylog.Debugf("Entering scontainer stage 1\n")

		if err := engine.CheckConfig(); err != nil {
			sylog.Fatalf("%s\n", err)
		}

		cconf.isInstance = C.uchar(bool2int(engine.IsRunAsInstance()))
		cconf.noNewPrivs = C.uchar(bool2int(engine.OciConfig.Process.NoNewPrivileges))

		cconf.uidMapping[0].containerID = C.uid_t(os.Getuid())
		cconf.uidMapping[0].hostID = C.uid_t(os.Getuid())
		cconf.uidMapping[0].size = 1
		cconf.gidMapping[0].containerID = C.gid_t(os.Getgid())
		cconf.gidMapping[0].hostID = C.gid_t(os.Getgid())
		cconf.gidMapping[0].size = 1

		if engine.OciConfig.Linux != nil {
			for _, namespace := range engine.OciConfig.Linux.Namespaces {
				switch namespace.Type {
				case specs.UserNamespace:
					cconf.nsFlags |= syscall.CLONE_NEWUSER
				case specs.IPCNamespace:
					cconf.nsFlags |= syscall.CLONE_NEWIPC
				case specs.UTSNamespace:
					cconf.nsFlags |= syscall.CLONE_NEWUTS
				case specs.PIDNamespace:
					cconf.nsFlags |= syscall.CLONE_NEWPID
				case specs.NetworkNamespace:
					cconf.nsFlags |= syscall.CLONE_NEWNET
				case specs.MountNamespace:
					cconf.nsFlags |= syscall.CLONE_NEWNS
				}
			}
		}
		jsonConf, _ := json.Marshal(engine.Common)
		cconf.jsonConfSize = C.uint(len(jsonConf))
		sylog.Debugf("jsonConfSize = %v\n", cconf.jsonConfSize)
		cconfPayload := C.GoBytes(unsafe.Pointer(cconf), C.sizeof_struct_cConfig)

		os.Stdout.Write(append(cconfPayload, jsonConf...))
		os.Exit(0)
	} else {
		if err := engine.StartProcess(); err != nil {
			sylog.Fatalf("%s\n", err)
		}
	}
}

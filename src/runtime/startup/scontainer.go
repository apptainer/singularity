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
	"github.com/singularityware/singularity/src/pkg/util/capabilities"
	"github.com/singularityware/singularity/src/runtime/engines"
)

func bool2int(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

// SContainer performs container startup
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

		if cconf.isSuid == 1 && engine.IsAllowSUID() == false {
			sylog.Fatalf("runtime engine %s doesn't allow SUID workflow", engine.EngineName)
		}

		if err := engine.PrepareConfig(); err != nil {
			sylog.Fatalf("%s\n", err)
		}

		cconf.isInstance = C.uchar(bool2int(engine.IsRunAsInstance()))
		cconf.noNewPrivs = C.uchar(bool2int(engine.OciConfig.Process.NoNewPrivileges))

		if engine.OciConfig.Linux != nil {
			for i, uid := range engine.OciConfig.Linux.UIDMappings {
				cconf.uidMapping[i].containerID = C.uid_t(uid.ContainerID)
				cconf.uidMapping[i].hostID = C.uid_t(uid.HostID)
				cconf.uidMapping[i].size = C.uint(uid.Size)
			}
			for i, gid := range engine.OciConfig.Linux.UIDMappings {
				cconf.gidMapping[i].containerID = C.gid_t(gid.ContainerID)
				cconf.gidMapping[i].hostID = C.gid_t(gid.HostID)
				cconf.gidMapping[i].size = C.uint(gid.Size)
			}
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
				case specs.CgroupNamespace:
					cconf.nsFlags |= 0x2000000
				}
			}
		}
		if engine.OciConfig.Process != nil && engine.OciConfig.Process.Capabilities != nil {
			var caps uint64

			for _, v := range engine.OciConfig.Process.Capabilities.Permitted {
				caps |= (1 << capabilities.Map[v].Value)
			}
			cconf.capPermitted = C.ulonglong(caps)

			caps = 0
			for _, v := range engine.OciConfig.Process.Capabilities.Effective {
				caps |= (1 << capabilities.Map[v].Value)
			}
			cconf.capEffective = C.ulonglong(caps)

			caps = 0
			for _, v := range engine.OciConfig.Process.Capabilities.Inheritable {
				caps |= (1 << capabilities.Map[v].Value)
			}
			cconf.capInheritable = C.ulonglong(caps)

			caps = 0
			for _, v := range engine.OciConfig.Process.Capabilities.Bounding {
				caps |= (1 << capabilities.Map[v].Value)
			}
			cconf.capBounding = C.ulonglong(caps)

			caps = 0
			for _, v := range engine.OciConfig.Process.Capabilities.Ambient {
				caps |= (1 << capabilities.Map[v].Value)
			}
			cconf.capAmbient = C.ulonglong(caps)
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

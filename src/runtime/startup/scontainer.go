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
	"net"
	"os"
	"syscall"
	"unsafe"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/capabilities"
	"github.com/singularityware/singularity/src/pkg/util/fs/proc"
	"github.com/singularityware/singularity/src/runtime/engines"
)

func bool2int(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

// SContainer performs container startup
func SContainer(stage C.int, masterSocket C.int, config *C.struct_cConfig, jsonC *C.char) {
	var conn net.Conn
	var err error
	cconf := config

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

		if err := engine.PrepareConfig(conn); err != nil {
			sylog.Fatalf("%s\n", err)
		}

		cconf.isInstance = C.uchar(bool2int(engine.IsRunAsInstance()))
		cconf.noNewPrivs = C.uchar(bool2int(engine.OciConfig.Process.NoNewPrivileges))

		mntNs := false

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
				var pid uint

				pid = 0
				if namespace.Path != "" {
					if pid, err = proc.ExtractPid(namespace.Path); err != nil {
						sylog.Fatalf("failed to determine pid for proc path %s: %s", namespace.Path, err)
					}
				}

				switch namespace.Type {
				case specs.UserNamespace:
					cconf.nsFlags |= syscall.CLONE_NEWUSER
					cconf.userPid = C.pid_t(pid)
				case specs.IPCNamespace:
					cconf.nsFlags |= syscall.CLONE_NEWIPC
					cconf.ipcPid = C.pid_t(pid)
				case specs.UTSNamespace:
					cconf.nsFlags |= syscall.CLONE_NEWUTS
					cconf.utsPid = C.pid_t(pid)
				case specs.PIDNamespace:
					cconf.nsFlags |= syscall.CLONE_NEWPID
					cconf.pidPid = C.pid_t(pid)
				case specs.NetworkNamespace:
					cconf.nsFlags |= syscall.CLONE_NEWNET
					cconf.netPid = C.pid_t(pid)
				case specs.MountNamespace:
					cconf.nsFlags |= syscall.CLONE_NEWNS
					cconf.mntPid = C.pid_t(pid)
					mntNs = true
				case specs.CgroupNamespace:
					cconf.nsFlags |= 0x2000000
					cconf.cgroupPid = C.pid_t(pid)
				}
			}
		}

		if !mntNs {
			if err := engine.OciConfig.AddOrReplaceLinuxNamespace("mount", ""); err != nil {
				sylog.Fatalf("failed to add mount namespace: %s", err)
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
		if err := engine.StartProcess(conn); err != nil {
			sylog.Fatalf("%s\n", err)
		}
	}
}

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
	"os/signal"
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
func SContainer(stage C.int, socket C.int, rpcSocket C.int, config *C.struct_cConfig, jsonC *C.char) {
	rpcfd := rpcSocket

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
		/* wait childs process */
		rpcChild := make(chan os.Signal, 1)
		signal.Notify(rpcChild, syscall.SIGCHLD)

		rpcSocket := os.NewFile(uintptr(rpcfd), "rpc")

		if stage == 3 {
			conn, err := net.FileConn(rpcSocket)
			rpcSocket.Close()
			if err != nil {
				sylog.Fatalf("socket communication error: %s\n", err)
			}

			// send "creating" status notification to smaster
			if err := engine.CreateContainer(conn); err != nil {
				sylog.Fatalf("%s\n", err)
			}
			// send "created" status notification to smaster
			os.Exit(0)
		}

		if err := engine.PrestartProcess(); err != nil {
			sylog.Fatalf("container setup failed: %s\n", err)
		}

		code := 0
		rpcSocket.Close()

		var status syscall.WaitStatus
	sigloop:
		for {
			if cconf.mntPid != 0 {
				break sigloop
			}
			select {
			case _ = <-rpcChild:
				/*
				 * waiting 2 childs signal there, since Linux can merge signals, we wait for all childs
				 * when first SIGCHLD received
				 */
				for {
					pid, err := syscall.Wait4(-1, &status, syscall.WNOHANG|syscall.WUNTRACED, nil)
					if err == syscall.ECHILD {
						/* no more childs */
						signal.Stop(rpcChild)
						close(rpcChild)
						break sigloop
					}
					if pid > 0 {
						code += status.ExitStatus()
					}
				}
			}
		}
		if code != 0 {
			sylog.Fatalf("container setup failed\n")
		}

		/* force close on exec on socket file descriptor to distinguish an exec success and error */
		_, _, errsys := syscall.RawSyscall(syscall.SYS_FCNTL, uintptr(socket), syscall.F_SETFD, syscall.FD_CLOEXEC)
		if errsys != 0 {
			sylog.Fatalf("set close-on-exec failed\n")
		}

		if err := engine.StartProcess(); err != nil {
			sylog.Fatalf("%s\n", err)
		}
	}
}

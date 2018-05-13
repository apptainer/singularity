/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package main

/*
#include <sys/types.h>
#include "startup/wrapper.h"
*/
// #cgo CFLAGS: -I../../c -I../../c/lib
// #cgo LDFLAGS: -L../../../../builddir/lib/ -lruntime -luuid
import "C"

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"unsafe"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/runtime/workflows"
)

func bool2int(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

//export SContainer
func SContainer(stage C.int, socket C.int, rpc_socket C.int, sruntime *C.char, config *C.struct_cConfig, jsonC *C.char) {
	rpcfd := rpc_socket

	cconf := config

	runtimeName := C.GoString(sruntime)

	/* get json configuration */
	jsonBytes := C.GoBytes(unsafe.Pointer(jsonC), C.int(cconf.jsonConfSize))

	engine, err := workflows.NewRuntimeEngine(runtimeName, jsonBytes)
	if err != nil {
		log.Fatalln(err)
	}

	if stage == 1 {
		if err := engine.CheckConfig(); err != nil {
			log.Fatalln(err)
		}

		cconf.isInstance = C.uchar(bool2int(engine.IsRunAsInstance()))
		cconf.noNewPrivs = C.uchar(bool2int(engine.OciConfig.RuntimeOciSpec.Process.NoNewPrivileges))

		cconf.uidMapping[0].containerID = C.uid_t(os.Getuid())
		cconf.uidMapping[0].hostID = C.uid_t(os.Getuid())
		cconf.uidMapping[0].size = 1
		cconf.gidMapping[0].containerID = C.gid_t(os.Getgid())
		cconf.gidMapping[0].hostID = C.gid_t(os.Getgid())
		cconf.gidMapping[0].size = 1

		for _, namespace := range engine.OciConfig.RuntimeOciSpec.Linux.Namespaces {
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

		jsonConf, _ := engine.GetConfig()
		cconf.jsonConfSize = C.uint(len(jsonConf))
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
				log.Fatalln("communication error")
			}

			// send "creating" status notification to smaster
			if err := engine.CreateContainer(conn); err != nil {
				os.Exit(1)
			}
			// send "created" status notification to smaster
			os.Exit(0)
		}

		if err := engine.PrestartProcess(); err != nil {
			log.Fatalln("Container setup failed:", err)
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
			log.Fatalln("Container setup failed:", code)
		}

		/* force close on exec on socket file descriptor to distinguish an exec success and error */
		_, _, errsys := syscall.RawSyscall(syscall.SYS_FCNTL, uintptr(socket), syscall.F_SETFD, syscall.FD_CLOEXEC)
		if errsys != 0 {
			log.Fatalln("set close-on-exec failed:", errsys)
		}

		if err := engine.StartProcess(); err != nil {
			log.Fatalln(err)
		}
	}
}

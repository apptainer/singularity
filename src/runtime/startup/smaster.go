// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

/*
#include "c/wrapper.c"
*/
// #cgo CFLAGS: -I..
import "C"

import (
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/singularityware/singularity/src/runtime/engines/common/config/wrapper"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines"
)

// SMaster initializes a runtime engine and runs it
func SMaster(socket int, masterSocket int, wrapperConfig *wrapper.Config, jsonBytes []byte) {
	var fatal error
	var status syscall.WaitStatus

	fatalChan := make(chan error, 1)
	ppid := os.Getppid()

	containerPid := wrapperConfig.GetContainerPid()

	engine, err := engines.NewEngine(jsonBytes)
	if err != nil {
		sylog.Fatalf("failed to initialize runtime: %s\n", err)
	}

	go func() {
		comm := os.NewFile(uintptr(socket), "socket")
		conn, err := net.FileConn(comm)
		if err != nil {
			fatalChan <- fmt.Errorf("failed to copy unix socket descriptor: %s", err)
			return
		}
		comm.Close()

		runtime.LockOSThread()
		if err := engine.CreateContainer(containerPid, conn); err != nil {
			fatalChan <- fmt.Errorf("container creation failed: %s", err)
		} else {
			conn.Close()
		}
		runtime.Goexit()
	}()

	if wrapperConfig.GetInstance() {
		go func() {
			data := make([]byte, 1)

			comm := os.NewFile(uintptr(masterSocket), "master-socket")
			conn, err := net.FileConn(comm)
			comm.Close()

			_, err = conn.Read(data)
			if err == io.EOF {
				/* sleep a bit to see if child exit */
				time.Sleep(100 * time.Millisecond)
				if os.Getppid() == ppid {
					syscall.Kill(ppid, syscall.SIGUSR1)
				}
			}
			conn.Close()
		}()
	}

	go func() {
		status, err = engine.MonitorContainer(containerPid)
		fatalChan <- err
	}()

	fatal = <-fatalChan

	runtime.LockOSThread()
	if err := engine.CleanupContainer(); err != nil {
		sylog.Errorf("container cleanup failed: %s", err)
	}
	runtime.UnlockOSThread()

	if fatal != nil {
		sylog.Fatalf("%s", fatal)
	}

	if status.Exited() {
		sylog.Debugf("Child exited with exit status %d", status.ExitStatus())
		if wrapperConfig.GetInstance() {
			if status.ExitStatus() != 0 {
				if os.Getppid() == ppid {
					syscall.Kill(ppid, syscall.SIGUSR2)
				}
				sylog.Fatalf("failed to spawn instance")
			}
			if os.Getppid() == ppid {
				syscall.Kill(ppid, syscall.SIGUSR1)
			}
		}
		os.Exit(status.ExitStatus())
	} else if status.Signaled() {
		sylog.Debugf("Child exited due to signal %d", status.Signal())
		syscall.Kill(os.Getpid(), status.Signal())
		os.Exit(1)
	}
}

func main() {
	loglevel := os.Getenv("SINGULARITY_MESSAGELEVEL")

	os.Clearenv()

	if loglevel != "" {
		if os.Setenv("SINGULARITY_MESSAGELEVEL", loglevel) != nil {
			sylog.Warningf("can't restore SINGULARITY_MESSAGELEVEL environment variable")
		}
	}

	cconf := unsafe.Pointer(&C.config)
	wrapperConfig := wrapper.NewConfig(wrapper.CConfig(cconf))
	jsonBytes := C.GoBytes(unsafe.Pointer(C.json_stdin), C.int(wrapperConfig.GetJSONConfSize()))

	switch C.execute {
	case C.SCONTAINER_STAGE1:
		sylog.Verbosef("Execute scontainer stage 1\n")
		SContainer(int(C.SCONTAINER_STAGE1), int(C.master_socket[1]), wrapperConfig, jsonBytes)
	case C.SCONTAINER_STAGE2:
		sylog.Verbosef("Execute scontainer stage 2\n")
		SContainer(int(C.SCONTAINER_STAGE2), int(C.master_socket[1]), wrapperConfig, jsonBytes)
	case C.SMASTER:
		sylog.Verbosef("Execute smaster process\n")
		SMaster(int(C.rpc_socket[0]), int(C.master_socket[0]), wrapperConfig, jsonBytes)
	case C.RPC_SERVER:
		sylog.Verbosef("Serve RPC requests\n")
		RPCServer(int(C.rpc_socket[1]), C.GoString(C.sruntime))

		sylog.Verbosef("Execute scontainer stage 2\n")
		C.prepare_scontainer_stage(C.SCONTAINER_STAGE2)
		SContainer(int(C.SCONTAINER_STAGE2), int(C.master_socket[1]), wrapperConfig, jsonBytes)
	}
	sylog.Fatalf("You should not be there\n")
}

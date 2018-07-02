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
	"io"
	"net"
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines"
)

// SMaster initializes a runtime engine and runs it
//export SMaster
func SMaster(socket C.int, instanceSocket C.int, config *C.struct_cConfig, jsonC *C.char) {
	containerPid := int(config.containerPid)
	jsonBytes := C.GoBytes(unsafe.Pointer(jsonC), C.int(config.jsonConfSize))

	engine, err := engines.NewEngine(jsonBytes)
	if err != nil {
		if err := engine.CleanupContainer(); err != nil {
			sylog.Errorf("container cleanup failed: %s", err)
		}
		sylog.Fatalf("failed to initialize runtime: %s\n", err)
	}

	go func() {
		comm := os.NewFile(uintptr(socket), "socket")
		conn, err := net.FileConn(comm)
		if err != nil {
			sylog.Fatalf("Failed to copy unix socket descriptor")
		}
		comm.Close()

		runtime.LockOSThread()
		defer conn.Close()
		if err := engine.CreateContainer(containerPid, conn); err != nil {
			sylog.Errorf("container creation failed: %s", err)
		}
		runtime.Goexit()
	}()

	if engine.IsRunAsInstance() {
		go func() {
			data := make([]byte, 1)

			comm := os.NewFile(uintptr(instanceSocket), "instance-socket")
			conn, err := net.FileConn(comm)
			comm.Close()

			_, err = conn.Read(data)
			if err == io.EOF {
				/* sleep a bit to see if child exit */
				time.Sleep(100 * time.Millisecond)
				syscall.Kill(syscall.Getppid(), syscall.SIGUSR1)
			}
			conn.Close()
		}()
	}

	status, err := engine.MonitorContainer(containerPid)
	if err != nil {
		sylog.Errorf("%s", err)
	}

	if err := engine.CleanupContainer(); err != nil {
		sylog.Errorf("container cleanup failed: %s", err)
	}

	if status.Exited() {
		sylog.Debugf("Child exited with exit status %d", status.ExitStatus())
		if engine.IsRunAsInstance() {
			if status.ExitStatus() != 0 {
				syscall.Kill(syscall.Getppid(), syscall.SIGUSR2)
				sylog.Fatalf("failed to spawn instance")
			}
		}
		os.Exit(status.ExitStatus())
	} else if status.Signaled() {
		sylog.Debugf("Child exited due to signal %d", status.Signal())
		syscall.Kill(os.Getpid(), status.Signal())
	}
}

func main() {}

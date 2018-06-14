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
	"os/signal"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines"
)

func runAsInstance(socket uintptr, engine *engines.Engine) {
	data := make([]byte, 1)

	comm := os.NewFile(socket, "instance-socket")
	conn, err := net.FileConn(comm)
	comm.Close()

	n, err := conn.Read(data)
	if n == 0 && err != io.EOF {
		syscall.Kill(syscall.Getppid(), syscall.SIGUSR2)
		if err := engine.CleanupContainer(); err != nil {
			sylog.Errorf("container cleanup failed: %s", err)
		}
		sylog.Fatalf("failed to spawn instance")
	} else {
		/* sleep a bit to see if child exit */
		time.Sleep(100 * time.Millisecond)
		syscall.Kill(syscall.Getppid(), syscall.SIGUSR1)
	}
}

func handleChild(pid int, signal chan os.Signal, engine *engines.Engine) {
	var status syscall.WaitStatus

	for {
		select {
		case _ = <-signal:
			if wpid, err := syscall.Wait4(pid, &status, syscall.WNOHANG, nil); err != nil {
				sylog.Errorf("failed while waiting child: %s", err)
			} else if wpid != pid {
				continue
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
	}
}

// SMaster initializes a runtime engine and runs it
//export SMaster
func SMaster(socket C.int, instanceSocket C.int, netFd C.int, config *C.struct_cConfig, jsonC *C.char) {
	var wg sync.WaitGroup

	sigchld := make(chan os.Signal, 1)
	signal.Notify(sigchld, syscall.SIGCHLD)

	os.Setenv("PATH", "/bin:/sbin:/usr/bin:/usr/sbin")

	containerPid := int(config.containerPid)
	jsonBytes := C.GoBytes(unsafe.Pointer(jsonC), C.int(config.jsonConfSize))

	comm := os.NewFile(uintptr(socket), "socket")
	conn, err := net.FileConn(comm)
	comm.Close()

	engine, err := engines.NewEngine(jsonBytes)
	if err != nil {
		if err := engine.CleanupContainer(); err != nil {
			sylog.Errorf("container cleanup failed: %s", err)
		}
		sylog.Fatalf("failed to initialize runtime: %s\n", err)
	}

	if engine.IsRunAsInstance() {
		go runAsInstance(uintptr(instanceSocket), engine)
	}

	go handleChild(containerPid, sigchld, engine)

	if err := engine.CreateContainer(conn); err != nil {
		if err := engine.CleanupContainer(); err != nil {
			sylog.Errorf("container cleanup failed: %s", err)
		}
		sylog.Fatalf("%s", err)
	}

	wg.Add(1)
	go engine.MonitorContainer()
	wg.Wait()

	os.Exit(0)
}

func main() {}

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
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines"
)

// SMaster initializes a runtime engine and runs it
//export SMaster
func SMaster(socket C.int, instanceSocket C.int, netFd C.int, config *C.struct_cConfig, jsonC *C.char) {
	var wg sync.WaitGroup

	signals := make(chan os.Signal, 1)
	signal.Notify(signals)

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
		if err := engine.CreateContainer(conn); err != nil {
			if err := engine.CleanupContainer(); err != nil {
				sylog.Errorf("container cleanup failed: %s", err)
			}
			sylog.Fatalf("%s", err)
		} else {
			wg.Add(1)
			engine.MonitorContainer()
			wg.Done()
		}
		runtime.Goexit()
	}()

	if engine.IsRunAsInstance() {
		data := make([]byte, 1)

		comm := os.NewFile(uintptr(instanceSocket), "instance-socket")
		conn, err := net.FileConn(comm)
		comm.Close()

		_, err = conn.Read(data)
		if err == io.EOF {
			go func() {
				/* sleep a bit to see if child exit */
				time.Sleep(100 * time.Millisecond)
				syscall.Kill(syscall.Getppid(), syscall.SIGUSR1)
			}()
		}
		conn.Close()
	}

	for {
		s := <-signals
		switch s {
		case syscall.SIGCHLD:
			var status syscall.WaitStatus

			if wpid, err := syscall.Wait4(containerPid, &status, syscall.WNOHANG, nil); err != nil {
				sylog.Errorf("failed while waiting child: %s", err)
			} else if wpid != containerPid {
				continue
			}

			if err := engine.CleanupContainer(); err != nil {
				sylog.Errorf("container cleanup failed: %s", err)
			}

			wg.Wait()

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
		default:
			if err := engine.CleanupContainer(); err != nil {
				sylog.Errorf("container cleanup failed: %s", err)
			}

			wg.Wait()
		}
	}
}

func main() {}

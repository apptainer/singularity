// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
)

// SMaster initializes a runtime engine and runs it
func SMaster(socket int, masterSocket int, startupConfig *config.Startup, jsonBytes []byte) {
	var fatal error
	var status syscall.WaitStatus

	fatalChan := make(chan error, 1)
	ppid := os.Getppid()

	containerPid := startupConfig.GetContainerPid()

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

	if startupConfig.GetInstance() {
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
		if startupConfig.GetInstance() {
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

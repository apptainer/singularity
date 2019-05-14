// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package starter

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/sylabs/singularity/internal/pkg/runtime/engines"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/mainthread"
)

// Master initializes a runtime engine and runs it
func Master(rpcSocket, masterSocket int, isInstance bool, containerPid int, engine *engines.Engine) {
	var fatal error
	var status syscall.WaitStatus

	fatalChan := make(chan error, 1)
	ppid := os.Getppid()

	go func() {
		comm := os.NewFile(uintptr(rpcSocket), "socket")
		rpcConn, err := net.FileConn(comm)
		comm.Close()
		if err != nil {
			fatalChan <- fmt.Errorf("failed to copy unix socket descriptor: %s", err)
			return
		}

		runtime.LockOSThread()
		err = engine.CreateContainer(containerPid, rpcConn)
		if err != nil {
			fatalChan <- fmt.Errorf("container creation failed: %s", err)
		} else {
			rpcConn.Close()
		}

		runtime.Goexit()
	}()

	go func() {
		data := make([]byte, 1)
		comm := os.NewFile(uintptr(masterSocket), "master-socket")
		conn, err := net.FileConn(comm)
		comm.Close()
		if err != nil {
			fatalChan <- fmt.Errorf("failed to create master connection: %s", err)
		}
		defer conn.Close()

		// special path for engines which needs to stop before executing
		// container process
		if obj, ok := engine.EngineOperations.(interface {
			PreStartProcess(int, net.Conn, chan error) error
		}); ok {
			n, err := conn.Read(data)
			if (err != nil && err != io.EOF) || n == 0 || data[0] == 'f' {
				if isInstance && os.Getppid() == ppid {
					syscall.Kill(ppid, syscall.SIGUSR2)
				}
				return
			}
			if err := obj.PreStartProcess(containerPid, conn, fatalChan); err != nil {
				fatalChan <- fmt.Errorf("pre start process failed: %s", err)
				return
			}
		}
		// wait container process execution, any error different from EOF
		// or any data send over master connection at this point means an
		// error occurred in StartProcess, just return by waiting error
		n, err := conn.Read(data)
		if (err != nil && err != io.EOF) || n > 0 {
			return
		}

		err = engine.PostStartProcess(containerPid)
		if err != nil {
			if isInstance && os.Getppid() == ppid {
				syscall.Kill(ppid, syscall.SIGUSR2)
			}
			fatalChan <- fmt.Errorf("post start process failed: %s", err)
			return
		}
		if isInstance {
			// sleep a bit to see if child exit
			time.Sleep(100 * time.Millisecond)
			if os.Getppid() == ppid {
				syscall.Kill(ppid, syscall.SIGUSR1)
			}
		}
	}()

	go func() {
		var err error

		// catch all signals
		signals := make(chan os.Signal, 1)
		signal.Notify(signals)

		status, err = engine.MonitorContainer(containerPid, signals)
		fatalChan <- err
	}()

	fatal = <-fatalChan

	runtime.LockOSThread()
	if err := engine.CleanupContainer(fatal, status); err != nil {
		sylog.Errorf("container cleanup failed: %s", err)
	}
	runtime.UnlockOSThread()

	// reset signal handlers
	signal.Reset()

	if fatal != nil {
		if isInstance && os.Getppid() == ppid {
			syscall.Kill(ppid, syscall.SIGUSR2)
		}
		syscall.Kill(containerPid, syscall.SIGKILL)
		sylog.Fatalf("%s", fatal)
	}

	exitCode := 0

	if status.Signaled() {
		s := status.Signal()
		sylog.Debugf("Child exited due to signal %d", s)
		if isInstance && os.Getppid() == ppid {
			syscall.Kill(ppid, syscall.SIGUSR2)
		}
		exitCode = 128 + int(s)
	} else if status.Exited() {
		sylog.Debugf("Child exited with exit status %d", status.ExitStatus())
		if isInstance && os.Getppid() == ppid {
			if status.ExitStatus() != 0 {
				syscall.Kill(ppid, syscall.SIGUSR2)
				sylog.Fatalf("failed to spawn instance")
			} else {
				syscall.Kill(ppid, syscall.SIGUSR1)
			}
		}
		exitCode = status.ExitStatus()
	}

	// mimic signal
	if exitCode > 128 && exitCode < 128+int(syscall.SIGUNUSED) {
		mainthread.Execute(func() {
			syscall.Kill(os.Getpid(), syscall.Signal(exitCode-128))
		})
	}

	// if previous signal didn't interrupt process
	os.Exit(exitCode)
}

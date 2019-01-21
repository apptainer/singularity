// Copyright (c) 2018, Sylabs Inc. All rights reserved.
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
	"unsafe"

	"github.com/sylabs/singularity/internal/pkg/runtime/engines"
	"github.com/sylabs/singularity/internal/pkg/sylog"
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

		n, err := conn.Read(data)
		if err != nil && err != io.EOF {
			if isInstance && os.Getppid() == ppid {
				syscall.Kill(ppid, syscall.SIGUSR2)
			}
			fatalChan <- fmt.Errorf("failed to start process: %s", err)
			return
		}

		// special path for engines which needs to stop before executing
		// container process
		if n != 0 {
			if obj, ok := engine.EngineOperations.(interface {
				PreStartProcess(int, net.Conn, chan error) error
			}); ok {
				if err := obj.PreStartProcess(containerPid, conn, fatalChan); err != nil {
					fatalChan <- fmt.Errorf("pre start process failed: %s", err)
					return
				}
			}
		}

		err = engine.PostStartProcess(containerPid)
		if err != nil {
			if isInstance && os.Getppid() == ppid {
				syscall.Kill(ppid, syscall.SIGUSR2)
			}
			fatalChan <- fmt.Errorf("post start process failed: %s", err)
			return
		}
		if n == 0 && isInstance {
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

	if !isInstance {
		pgrp := syscall.Getpgrp()
		tcpgrp := 0

		if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, 1, uintptr(syscall.TIOCGPGRP), uintptr(unsafe.Pointer(&tcpgrp))); err == 0 {
			if tcpgrp > 0 && pgrp != tcpgrp {
				signal.Ignore(syscall.SIGTTOU)

				if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, 1, uintptr(syscall.TIOCSPGRP), uintptr(unsafe.Pointer(&pgrp))); err != 0 {
					sylog.Errorf("failed to set crontrolling terminal group: %s", err.Error())
				}
			}
		}
	}

	if fatal != nil {
		if isInstance {
			if os.Getppid() == ppid {
				syscall.Kill(ppid, syscall.SIGUSR2)
			}
		}
		syscall.Kill(containerPid, syscall.SIGKILL)
		sylog.Fatalf("%s", fatal)
	}

	if status.Signaled() {
		sylog.Debugf("Child exited due to signal %d", status.Signal())
		if isInstance && os.Getppid() == ppid {
			syscall.Kill(ppid, syscall.SIGUSR2)
		}
		os.Exit(128 + int(status.Signal()))
	} else if status.Exited() {
		sylog.Debugf("Child exited with exit status %d", status.ExitStatus())
		if isInstance {
			if status.ExitStatus() != 0 {
				if os.Getppid() == ppid {
					syscall.Kill(ppid, syscall.SIGUSR2)
					sylog.Fatalf("failed to spawn instance")
				}
			}
			if os.Getppid() == ppid {
				syscall.Kill(ppid, syscall.SIGUSR1)
			}
		}
		os.Exit(status.ExitStatus())
	}
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

/*
#include "c/starter.c"
*/
// #cgo CFLAGS: -I..
import "C"

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

	"github.com/sylabs/singularity/src/pkg/sylog"
	"github.com/sylabs/singularity/src/pkg/util/mainthread"
	"github.com/sylabs/singularity/src/runtime/engines"
	"github.com/sylabs/singularity/src/runtime/engines/config/starter"
)

// SMaster initializes a runtime engine and runs it
func SMaster(rpcSocket, masterSocket int, starterConfig *starter.Config, jsonBytes []byte) {
	var fatal error
	var status syscall.WaitStatus

	fatalChan := make(chan error, 1)
	ppid := os.Getppid()
	containerPid := starterConfig.GetContainerPid()
	engine, err := engines.NewEngine(jsonBytes)
	if err != nil {
		sylog.Fatalf("failed to initialize runtime: %s\n", err)
	}

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

		_, err = conn.Read(data)
		if err != nil && err != io.EOF {
			if starterConfig.GetInstance() && os.Getppid() == ppid {
				syscall.Kill(ppid, syscall.SIGUSR2)
			}
			fatalChan <- fmt.Errorf("failed to start process: %s", err)
			return
		}
		err = engine.PostStartProcess(containerPid)
		if err != nil {
			if starterConfig.GetInstance() && os.Getppid() == ppid {
				syscall.Kill(ppid, syscall.SIGUSR2)
			}
			fatalChan <- fmt.Errorf("post start process failed: %s", err)
			return
		}
		if starterConfig.GetInstance() {
			// sleep a bit to see if child exit
			time.Sleep(100 * time.Millisecond)
			if os.Getppid() == ppid {
				syscall.Kill(ppid, syscall.SIGUSR1)
			}
		}
	}()

	go func() {
		// catch all signals
		signals := make(chan os.Signal, 1)
		signal.Notify(signals)

		status, err = engine.MonitorContainer(containerPid, signals)
		fatalChan <- err
	}()
	fatal = <-fatalChan

	runtime.LockOSThread()
	if err := engine.CleanupContainer(); err != nil {
		sylog.Errorf("container cleanup failed: %s", err)
	}
	runtime.UnlockOSThread()

	if fatal != nil {
		if starterConfig.GetInstance() {
			if os.Getppid() == ppid {
				syscall.Kill(ppid, syscall.SIGUSR2)
			}
		}
		syscall.Kill(containerPid, syscall.SIGKILL)
		sylog.Fatalf("%s", fatal)
	}

	if status.Signaled() {
		sylog.Debugf("Child exited due to signal %d", status.Signal())
		if os.Getppid() == ppid {
			syscall.Kill(ppid, syscall.SIGUSR2)
		}
		syscall.Kill(syscall.Gettid(), syscall.SIGKILL)
	} else if status.Exited() {
		sylog.Debugf("Child exited with exit status %d", status.ExitStatus())
		if starterConfig.GetInstance() {
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

func startup() {
	loglevel := os.Getenv("SINGULARITY_MESSAGELEVEL")
	os.Clearenv()
	if loglevel != "" {
		if os.Setenv("SINGULARITY_MESSAGELEVEL", loglevel) != nil {
			sylog.Warningf("can't restore SINGULARITY_MESSAGELEVEL environment variable")
		}
	}

	cconf := unsafe.Pointer(&C.config)
	starterConfig := starter.NewConfig(starter.CConfig(cconf))
	jsonBytes := C.GoBytes(unsafe.Pointer(C.json_stdin), C.int(starterConfig.GetJSONConfSize()))

	// free allocated buffer
	C.free(unsafe.Pointer(C.json_stdin))
	if unsafe.Pointer(C.nspath) != nil {
		C.free(unsafe.Pointer(C.nspath))
	}

	switch C.execute {
	case C.SCONTAINER_STAGE1:
		sylog.Verbosef("Execute scontainer stage 1\n")
		SContainer(int(C.SCONTAINER_STAGE1), int(C.master_socket[1]), starterConfig, jsonBytes)
	case C.SCONTAINER_STAGE2:
		sylog.Verbosef("Execute scontainer stage 2\n")
		mainthread.Execute(func() {
			SContainer(int(C.SCONTAINER_STAGE2), int(C.master_socket[1]), starterConfig, jsonBytes)
		})
	case C.SMASTER:
		sylog.Verbosef("Execute smaster process\n")
		SMaster(int(C.rpc_socket[0]), int(C.master_socket[0]), starterConfig, jsonBytes)
	case C.RPC_SERVER:
		sylog.Verbosef("Serve RPC requests\n")
		RPCServer(int(C.rpc_socket[1]), C.GoString(C.sruntime))
	}
	sylog.Fatalf("You should not be there\n")
}

func init() {
	// lock main thread for function execution loop
	runtime.LockOSThread()
	// this is mainly to reduce memory footprint
	runtime.GOMAXPROCS(1)
}

func main() {
	go startup()

	// run functions requiring execution in main thread
	for f := range mainthread.FuncChannel {
		f()
	}
}

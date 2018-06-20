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
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/singularityware/singularity/src/pkg/network"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines"
)

func runAsInstance(conn *os.File) {
	data := make([]byte, 1)

	n, err := conn.Read(data)
	if n == 0 && err != io.EOF {
		os.Exit(1)
	} else {
		/* sleep a bit to see if child exit */
		time.Sleep(100 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
	}
}

func handleChild(pid int, signal chan os.Signal, engine *engines.Engine) {
	var status syscall.WaitStatus

	time.Sleep(200 * time.Millisecond)
	/* hold a reference to container network namespace for cleanup */
	f, err := os.Open("/proc/" + strconv.Itoa(pid) + "/ns/net")
	if err != nil {
		sylog.Fatalf("can't open network namespace: %s\n", err)
	}
	nspath := fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), f.Fd())
	list, err := network.NewNetworkList([]string{"bridge"}, strconv.Itoa(pid), nspath, nil)
	if err != nil {
		sylog.Fatalf("%s", err)
	}
	if err := list.AddNetworkArgs([]string{"bridge:portmap=8080:80/tcp"}); err != nil {
		sylog.Errorf("%s", err)
	}
	os.Setenv("PATH", "/bin:/sbin:/usr/bin:/usr/sbin")
	if err := list.SetupNetworks(); err != nil {
		sylog.Fatalf("%s", err)
	}

	for {
		select {
		case _ = <-signal:
			wpid, _ := syscall.Wait4(pid, &status, syscall.WNOHANG, nil)
			if wpid != pid {
				continue
			}

			sylog.Debugf("Cleanup container")
			if err := engine.CleanupContainer(); err != nil {
				sylog.Errorf("container cleanup failed: %s", err)
			}
			if err := list.CleanupNetworks(); err != nil {
				sylog.Errorf("%s", err)
			}

			if status.Exited() {
				sylog.Debugf("Child exited with exit status %d", status.ExitStatus())
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
func SMaster(socket C.int, config *C.struct_cConfig, jsonC *C.char) {
	var wg sync.WaitGroup

	sigchld := make(chan os.Signal, 1)
	signal.Notify(sigchld, syscall.SIGCHLD)

	containerPid := int(config.containerPid)
	jsonBytes := C.GoBytes(unsafe.Pointer(jsonC), C.int(config.jsonConfSize))

	comm := os.NewFile(uintptr(socket), "socket")

	engine, err := engines.NewEngine(jsonBytes)

	if err != nil {
		sylog.Fatalf("failed to initialize runtime: %s\n", err)
	}

	wg.Add(1)
	go handleChild(containerPid, sigchld, engine)

	wg.Add(1)
	go engine.MonitorContainer()

	if engine.IsRunAsInstance() {
		wg.Add(1)
		go runAsInstance(comm)
	}

	wg.Wait()
	os.Exit(0)
}

func main() {}

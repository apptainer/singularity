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
import "C"

import (
	"io"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/singularityware/singularity/src/pkg/sylog"
	runtime "github.com/singularityware/singularity/src/pkg/workflows"
	internalRuntime "github.com/singularityware/singularity/src/runtime/workflows"
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

func handleChild(pid int, child chan os.Signal, engine *runtime.RuntimeEngine) {
	var status syscall.WaitStatus

	select {
	case _ = <-child:
		syscall.Wait4(pid, &status, syscall.WNOHANG, nil)

		//		engine.CleanupContainer()
		/*
		 * see https://github.com/opencontainers/runtime-spec/blob/master/runtime.md#lifecycle
		 * we will run step 8/9 there
		 */

		os.Exit(status.ExitStatus())
	}
}

// SMaster initializes a runtime engine and runs it
//export SMaster
func SMaster(socket C.int, sruntime *C.char, config *C.struct_cConfig, jsonC *C.char) {
	var wg sync.WaitGroup

	sigchild := make(chan os.Signal, 1)
	signal.Notify(sigchild, syscall.SIGCHLD)

	containerPid := int(config.containerPid)
	runtimeName := C.GoString(sruntime)
	jsonBytes := C.GoBytes(unsafe.Pointer(jsonC), C.int(config.jsonConfSize))

	comm := os.NewFile(uintptr(socket), "socket")

	/* hold a reference to container network namespace for cleanup */
	_, err := os.Open("/proc/" + strconv.Itoa(containerPid) + "/ns/net")
	if err != nil {
		sylog.Fatalf("can't open network namespace: %s\n", err)
	}

	engine, err := internalRuntime.NewRuntimeEngine(runtimeName, jsonBytes)
	if err != nil {
		sylog.Fatalf("failed to initialize runtime: %s\n", err)
	}

	wg.Add(1)
	go handleChild(containerPid, sigchild, nil) //engine)

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

/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	internalRuntime "github.com/singularityware/singularity/internal/pkg/runtime"
	runtime "github.com/singularityware/singularity/pkg/runtime"
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
	case _ = (<-child):
		syscall.Wait4(pid, &status, syscall.WNOHANG, nil)

		engine.CleanupContainer()
		/*
		 * see https://github.com/opencontainers/runtime-spec/blob/master/runtime.md#lifecycle
		 * we will run step 8/9 there
		 */

		os.Exit(status.ExitStatus())
	}
}

func main() {
	var wg sync.WaitGroup

	sigchild := make(chan os.Signal, 1)
	signal.Notify(sigchild, syscall.SIGCHLD)

	tmp, ok := os.LookupEnv("SMASTER_CONTAINER_PID")
	if !ok {
		log.Fatalln("SMASTER_CONTAINER_PID environment variable isn't set")
	}
	containerPid, _ := strconv.Atoi(tmp)

	tmp, ok = os.LookupEnv("SMASTER_SOCKET")
	if !ok {
		log.Fatalln("SMASTER_SOCKET environment variable isn't set")
	}
	socket, _ := strconv.Atoi(tmp)

	tmp, ok = os.LookupEnv("SRUNTIME")
	if !ok {
		log.Fatalln("SRUNTIME environment variable isn't set")
	}
	runtimeName := tmp

	/* hold a reference to container network namespace for cleanup */
	_, err := os.Open("/proc/" + strconv.Itoa(containerPid) + "/ns/net")
	if err != nil {
		log.Fatalln("can't open network namespace:", err)
	}

	comm := os.NewFile(uintptr(socket), "socket")
	bytes, err := ioutil.ReadAll(comm)
	if err != nil {
		log.Fatalln("smaster read configuration failed", err)
	}

	engine, err := internalRuntime.NewRuntimeEngine(runtimeName, bytes)
	if err != nil {
		log.Fatalln("failed to initialize runtime:", err)
	}

	wg.Add(1)
	go handleChild(containerPid, sigchild, engine)

	if engine.IsRunAsInstance() {
		wg.Add(1)
		go runAsInstance(comm)
	}

	engine.MonitorContainer()

	wg.Wait()
	os.Exit(0)
}

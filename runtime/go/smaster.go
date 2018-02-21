/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package main

import (
	"encoding/json"
	_ "fmt"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

func instance(conn net.Conn, spec specs.Spec) {
	data := make([]byte, 1)
	conn.SetDeadline(time.Time{})

	n, err := conn.Read(data)
	if n == 0 && err != io.EOF {
		os.Exit(1)
	} else {
		/* sleep a bit to see if child exit */
		time.Sleep(100 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
	}
}

func handle_child(pid int, child chan os.Signal, spec specs.Spec) {
	var status syscall.WaitStatus
	userNS := false

	for _, namespace := range spec.Linux.Namespaces {
		switch namespace.Type {
		case specs.UserNamespace:
			userNS = true
		}
	}

	/* hold a reference to container network namespace for cleanup */
	if userNS == false {
		netns, err := os.Open("/proc/" + strconv.Itoa(pid) + "/ns/net")
		if err != nil {
			log.Fatalln("can't open network namespace:", err)
		}
		_ = netns
	}

	select {
	case _ = (<-child):
		syscall.Wait4(pid, &status, syscall.WNOHANG, nil)

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

	pid, _ := strconv.Atoi(os.Args[1])
	socket, _ := strconv.Atoi(os.Args[2])

	comm := os.NewFile(uintptr(socket), "")

	conn, _ := net.FileConn(comm)
	comm.Close()
	conn.SetDeadline(time.Now().Add(1 * time.Second))

	var spec specs.Spec

	decoder := json.NewDecoder(conn)
	err := decoder.Decode(&spec)
	if err != nil {
		log.Fatalln("smaster read configuration failed", err)
	}

	wg.Add(1)
	go handle_child(pid, sigchild, spec)
	/*
		if jconf.IsInstance {
			wg.Add(1)
			go instance(conn, spec)

			if jconf.UserNS {
				fmt.Printf("To join instance: nsenter -t %d -U --preserve-credentials -m -p -u -i -n -r /bin/sh\n", pid)
			} else {
				fmt.Printf("To join instance: sudo nsenter -t %d -m -p -u -i -n -r /bin/sh\n", pid)
			}
		}
	*/
	wg.Wait()
	os.Exit(0)
}

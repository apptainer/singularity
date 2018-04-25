/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package main

import "C"

import (
	"log"
	"net"
	"os"

	"github.com/singularityware/singularity/src/runtime/workflows/rpc"
)

// If you update the CGo exported functions YOU MUST UPDATE
// the src/runtime/c/startup/librpc.h header file by compiling
// the project and copying buildtree/librpc.h

//export RPCServer
func RPCServer(socket int) int {
	tmp, ok := os.LookupEnv("SRUNTIME")
	if !ok {
		log.Fatalln("SRUNTIME environment variable isn't set")
	}
	runtime := tmp

	comm := os.NewFile(uintptr(socket), "unix")

	conn, err := net.FileConn(comm)
	if err != nil {
		log.Fatalln("communication error")
	}
	comm.Close()

	rpc.ServeRuntimeEngineRequests(runtime, conn)

	return 0
}

func main() {}

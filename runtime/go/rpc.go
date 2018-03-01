/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package main

import "C"

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"rpc/server"
)

//export RpcServer
func RpcServer(socket int) int {
	comm := os.NewFile(uintptr(socket), "unix")

	conn, err := net.FileConn(comm)
	if err != nil {
		fmt.Println("communication error")
        return 1
	}
	comm.Close()

	rpcOps := new(server.RpcOps)
	rpc.RegisterName("Privileged", rpcOps)
	rpc.ServeConn(conn)

    return 0
}

func main() {}

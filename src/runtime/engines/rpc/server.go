// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package rpc

import (
	"net"
	"net/rpc"

	singularityConfig "github.com/singularityware/singularity/src/runtime/engines/singularity/config"
	singularityRpcServer "github.com/singularityware/singularity/src/runtime/engines/singularity/rpc/server"
)

var rpcServerMethods map[string]interface{}

// Register RPC server methods for runtime engine
func registerRuntimeRPCServerMethods(methods interface{}, name string) {
	if rpcServerMethods == nil {
		rpcServerMethods = make(map[string]interface{})
	}
	rpcServerMethods[name] = methods
}

// ServeRuntimeEngineRequests services runtime engine requests
func ServeRuntimeEngineRequests(name string, conn net.Conn) {
	methods := rpcServerMethods[name]
	rpc.RegisterName(name, methods)
	rpc.ServeConn(conn)
}

func init() {
	// register singularity RPC methods
	methods := new(singularityRpcServer.Methods)
	registerRuntimeRPCServerMethods(methods, singularityConfig.Name)
}

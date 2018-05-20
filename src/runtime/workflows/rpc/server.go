package rpc

import (
	"net"
	"net/rpc"

	singularityConfig "github.com/singularityware/singularity/src/runtime/workflows/workflows/singularity/config"
	singularityRpcServer "github.com/singularityware/singularity/src/runtime/workflows/workflows/singularity/rpc/server"
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

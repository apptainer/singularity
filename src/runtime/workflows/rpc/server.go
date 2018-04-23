package rpc

import (
	singularityConfig "github.com/singularityware/singularity/src/runtime/workflows/workflows/singularity/config"
	singularityRpcServer "github.com/singularityware/singularity/src/runtime/workflows/workflows/singularity/rpc/server"
	"net"
	"net/rpc"
)

var rpcServerMethods map[string]interface{}

// Register RPC server methods for runtime engine
func registerRuntimeRpcServerMethods(methods interface{}, name string) {
	if rpcServerMethods == nil {
		rpcServerMethods = make(map[string]interface{})
	}
	rpcServerMethods[name] = methods
}

func ServeRuntimeEngineRequests(name string, conn net.Conn) {
	methods := rpcServerMethods[name]
	rpc.RegisterName(name, methods)
	rpc.ServeConn(conn)
}

func init() {
	// register singularity RPC methods
	methods := new(singularityRpcServer.Methods)
	registerRuntimeRpcServerMethods(methods, singularityConfig.Name)
}

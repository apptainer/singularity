package rpc

import(
    "net"
    "net/rpc"
    singularityRpcServer "github.com/singularityware/singularity/internal/pkg/runtime/engine/singularity/rpc/server"
    singularityConfig "github.com/singularityware/singularity/internal/pkg/runtime/engine/singularity/config"
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

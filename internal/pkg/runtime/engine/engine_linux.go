// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package engine

import (
	"encoding/json"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config/starter"
)

// Engine is the combination of an Operations and a config.Common. The singularity
// startup routines (internal/app/starter/*) can spawn a container process from this type.
type Engine struct {
	Operations
	*config.Common
}

// Operations is an interface describing necessary operations to launch
// a container process. Some of them may be called with elevated privilege
// or the potential to escalate privileges. Refer to an individual method
// documentation for a detailed description of the context in which it is called.
type Operations interface {
	// Config returns a zero value of the current EngineConfig, which
	// depends on the implementation, used to populate the Common struct.
	// Since this method simply return a zero value of the concrete
	// EngineConfig, it does not matter whether or not there are any elevated
	// privileges during this call.
	Config() config.EngineConfig
	// InitConfig stores the parsed config.Common inside the Operations
	// implementation. Since this method simply stores config.Common,
	// it does not matter whether or not there are any elevated
	// privileges during this call.
	InitConfig(*config.Common)
	// PrepareConfig is called during stage1 to validate and prepare
	// container configuration. No additional privileges can be gained
	// as any of them are already dropped by the time PrepareConfig is called.
	PrepareConfig(*starter.Config) error
	// CreateContainer is called in master and does mount operations, etc... to
	// set up the container environment for the payload proc.
	CreateContainer(int, net.Conn) error
	// StartProcess is called in stage2 after waiting on RPC server exit. It is
	// responsible for exec'ing the payload proc in the container.
	StartProcess(net.Conn) error
	// PostStartProcess is called in master after successful execution of container process.
	PostStartProcess(int) error
	// MonitorContainer is called in master once the container proc has been spawned. It
	// will typically block until the container proc exists.
	MonitorContainer(int, chan os.Signal) (syscall.WaitStatus, error)
	// CleanupContainer is called in master after the MonitorContainer returns. It is responsible
	// for ensuring that the container has been properly torn down.
	CleanupContainer(error, syscall.WaitStatus) error
}

// getName returns the engine name set in JSON []byte configuration.
func getName(b []byte) string {
	engineName := struct {
		EngineName string `json:"engineName"`
	}{}
	if err := json.Unmarshal(b, &engineName); err != nil {
		return ""
	}
	return engineName.EngineName
}

// Get returns the engine described by the JSON []byte configuration.
func Get(b []byte) (*Engine, error) {
	engineName := getName(b)

	// ensure engine with given name is registered
	eOp, ok := registeredOperations[engineName]
	if !ok {
		return nil, fmt.Errorf("engine %q is not found", engineName)
	}

	// create empty Engine object with properly initialized EngineConfig && Operations
	e := &Engine{
		Operations: eOp,
		Common: &config.Common{
			EngineConfig: eOp.Config(),
		},
	}

	// parse received JSON configuration to specific EngineConfig
	if err := json.Unmarshal(b, e.Common); err != nil {
		return nil, fmt.Errorf("could not parse JSON configuration: %s", err)
	}
	e.InitConfig(e.Common)
	return e, nil
}

var (
	// registeredOperations contains a map relating an Engine name to a set
	// of operations provided by an engine.
	registeredOperations = make(map[string]Operations)

	// registerEngineRPCMethods contains a map relating an Engine name to a set
	// of RPC methods served by RPC server.
	registeredRPCMethods = make(map[string]interface{})
)

// ServeRPCRequests serves runtime engine RPC requests with
// corresponding registered engine methods.
func ServeRPCRequests(e *Engine, conn net.Conn) {
	methods, ok := registeredRPCMethods[e.EngineName]
	if ok {
		rpc.RegisterName(e.EngineName, methods)
		rpc.ServeConn(conn)
	}
}

// RegisterOperations registers engine operations for a runtime engine.
func RegisterOperations(name string, operations Operations) {
	registeredOperations[name] = operations
}

// RegisterRPCMethods registers engine RPC methods served by RPC server.
func RegisterRPCMethods(name string, methods interface{}) {
	registeredRPCMethods[name] = methods
}

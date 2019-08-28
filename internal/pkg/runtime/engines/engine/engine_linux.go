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
	"sync"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/starter"
)

// Engine is the combination of an Operations and a config.Common. The singularity
// startup routines (src/runtime/startup/*) can spawn a container process from this type
type Engine struct {
	Operations
	*config.Common
}

// Operations is an interface describing necessary operations to launch a container process.
type Operations interface {
	// Config returns the current EngineConfig, used to populate the Common struct
	Config() config.EngineConfig
	// InitConfig is responsible for storing the parse config.Common inside
	// the Operations implementation.
	InitConfig(*config.Common)
	// PrepareConfig is called in stage1 to validate and prepare container configuration.
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
	// CleanupContainer is called in master after the MontiorContainer returns. It is responsible
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

	// prevents concurrent writes for registeredOperations and registerEngineRPCMethods
	// during RegisterOperations and RegisterRPCMethods calls because there are mainly
	// called from multiple init() at import time.
	mutex sync.Mutex
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

// RegisterOperations registers engine operations for a runtime
// engine
func RegisterOperations(name string, operations Operations) {
	mutex.Lock()
	defer mutex.Unlock()
	registeredOperations[name] = operations
}

// RegisterRPCMethods registers engine RPC methods served by RPC
// server
func RegisterRPCMethods(name string, methods interface{}) {
	mutex.Lock()
	defer mutex.Unlock()
	registeredRPCMethods[name] = methods
}

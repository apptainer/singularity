// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package engines

import (
	"encoding/json"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/starter"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/imgbuild"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/rpc/server"
)

// Engine is the combination of an EngineOperations and a config.Common. The singularity
// startup routines (src/runtime/startup/*) can spawn a container process from this type
type Engine struct {
	EngineOperations
	*config.Common
}

// EngineOperations is an interface describing necessary operations to launch a container process.
type EngineOperations interface {
	// Config returns the current EngineConfig, used to populate the Common struct
	Config() config.EngineConfig
	// InitConfig is responsible for storing the parse config.Common inside
	// the EngineOperations implementation.
	InitConfig(*config.Common)
	// PrepareConfig is called in stage1 to validate and prepare container configuration.
	PrepareConfig(net.Conn, *starter.Config) error
	// CreateContainer is called in smaster and does mount operations, etc... to
	// set up the container environment for the payload proc.
	CreateContainer(int, net.Conn) error
	// StartProcess is called in stage2 after waiting on RPC server exit. It is
	// responsible for exec'ing the payload proc in the container.
	StartProcess(net.Conn) error
	// PostStartProcess is called in smaster after successful execution of container process.
	PostStartProcess(int) error
	// MonitorContainer is called in smaster once the container proc has been spawned. It
	// will typically block until the container proc exists.
	MonitorContainer(int, chan os.Signal) (syscall.WaitStatus, error)
	// CleanupContainer is called in smaster after the MontiorContainer returns. It is responsible
	// for ensuring that the container has been properly torn down.
	CleanupContainer() error
}

// NewEngine returns the engine described by the JSON []byte configuration.
func NewEngine(b []byte) (*Engine, error) {
	engineName := struct {
		EngineName string `json:"engineName"`
	}{}
	if err := json.Unmarshal(b, &engineName); err != nil {
		return nil, fmt.Errorf("engineName field is not found: %v", err)
	}

	// ensure engine with given name is registered
	eOp, ok := registeredEngineOperations[engineName.EngineName]
	if !ok {
		return nil, fmt.Errorf("engine %q is not found", engineName)
	}

	// create empty Engine object with properly initialized EngineConfig && EngineOperations
	e := &Engine{
		EngineOperations: eOp,
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
	registeredEngineOperations map[string]EngineOperations

	// registerEngineRPCMethods contains a map relating an Engine name to a set
	// of RPC methods served by RPC server
	registeredEngineRPCMethods map[string]interface{}
)

// ServeRuntimeEngineRequests serves runtime engine requests with corresponding registered engine methods.
func ServeRuntimeEngineRequests(name string, conn net.Conn) {
	methods := registeredEngineRPCMethods[name]
	rpc.RegisterName(name, methods)
	rpc.ServeConn(conn)
}

// Init initializes registered runtime engines
func Init() {
	registeredEngineOperations = make(map[string]EngineOperations)
	registeredEngineOperations[singularity.Name] = &singularity.EngineOperations{EngineConfig: singularity.NewConfig()}
	registeredEngineOperations[imgbuild.Name] = &imgbuild.EngineOperations{EngineConfig: &imgbuild.EngineConfig{}}

	// register singularity rpc methods
	methods := new(server.Methods)
	registeredEngineRPCMethods = make(map[string]interface{})
	registeredEngineRPCMethods[singularity.Name] = methods
	registeredEngineRPCMethods[imgbuild.Name] = methods

}

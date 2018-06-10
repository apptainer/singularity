// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package engines

import (
	"encoding/json"
	"net"
	"net/rpc"

	"github.com/singularityware/singularity/src/runtime/engines/common/config"
	singularity "github.com/singularityware/singularity/src/runtime/engines/singularity"
	singularityConfig "github.com/singularityware/singularity/src/runtime/engines/singularity/config"
	singularityRpcServer "github.com/singularityware/singularity/src/runtime/engines/singularity/rpc/server"
)

/*
// ContainerLauncher is a struct containing the unique combination of an Engine
// with a RuntimeConfig. Together, this unique combination can launch one container
// or potentially set of containers.
type ContainerLauncher struct {
	Engine
	*config.RuntimeConfig
}
*/

// Engine is
type Engine struct {
	EngineOperations
	*config.Common
}

// EngineOperations is an interface describing necessary runtime operations to launch a
// container process. An Engine *uses* a RuntimeConfig to *launch* a container.
type EngineOperations interface {
	// intialize configuration and return it/
	InitConfig(*config.Common)
	// call in stage1
	CheckConfig() error
	// are we run as instance
	IsRunAsInstance() bool
	// call in child stage2
	CreateContainer(rpcConn net.Conn) error
	// call in parent stage2 before waiting stage2 child
	PrestartProcess() error
	// call in parent stage2 after stage2 child exit
	StartProcess() error
	// call in smaster once container is created
	MonitorContainer() error
	// call in smaster for container cleanup
	CleanupContainer() error
}

func NewEngine(b []byte) (*Engine, error) {
	e := &Engine{
		Common: &config.Common{},
	}

	if err := json.Unmarshal(b, e); err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Engine) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, e.Common); err != nil {
		return err
	}

	e.EngineOperations = registeredEngineOperations[e.EngineName]
	e.InitConfig(e.Common)

	return nil
}

var registeredEngineOperations map[string]EngineOperations

func registerEngineOperations(e EngineOperations, name string) {
	registeredEngineOperations[name] = e
}

// registerEngineRPCMethods contains a map relating an Engine name to a set
// of RPC methods served by RPC server
var registeredEngineRPCMethods map[string]interface{}

// registerEngineRPCMethods registers RPC server methods for runtime engine
func registerEngineRPCMethods(methods interface{}, name string) {
	registeredEngineRPCMethods[name] = methods
}

// ServeRuntimeEngineRequests serves runtime engine requests with corresponding
// registered engine methods
func ServeRuntimeEngineRequests(name string, conn net.Conn) {
	methods := registeredEngineRPCMethods[name]
	rpc.RegisterName(name, methods)
	rpc.ServeConn(conn)
}

func init() {
	registeredEngineOperations = make(map[string]EngineOperations)
	registeredEngineRPCMethods = make(map[string]interface{})

	methods := new(singularityRpcServer.Methods)
	registerEngineOperations(&singularity.EngineOperations{EngineConfig: singularityConfig.NewSingularityConfig()}, "singularity")
	registerEngineRPCMethods(methods, singularityConfig.Name)
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package engines

import (
	"fmt"
	"net"
	"net/rpc"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
	singularity "github.com/singularityware/singularity/src/runtime/engines/singularity"
	singularityConfig "github.com/singularityware/singularity/src/runtime/engines/singularity/config"
	singularityRpcServer "github.com/singularityware/singularity/src/runtime/engines/singularity/rpc/server"
)

// ContainerLauncher is a struct containing the unique combination of an Engine
// with a RuntimeConfig. Together, this unique combination can launch one container
// or potentially set of containers.
type ContainerLauncher struct {
	Engine
	*config.RuntimeConfig
}

// Engine is an interface describing necessary runtime operations to launch a
// container process. An Engine *uses* a RuntimeConfig to *launch* a container.
type Engine interface {
	// intialize configuration and return it/
	InitConfig() *config.RuntimeConfig
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

// registeredEngines contains a map relating an Engine name to a ContainerLauncher
// created with a default config. A new ContainerLauncher can later override the config
// to enable different container process launchings
var registeredEngines map[string]*ContainerLauncher

// registerEngine is used to register a specific engine in the registeredEngines
// map. This should be called from the init() function of a package implementing
// a data type satisfying the Engine interface
func registerEngine(e Engine, name string) {
	l := &ContainerLauncher{
		Engine:        e,
		RuntimeConfig: e.InitConfig(),
	}

	registeredEngines[name] = l
	if l.RuntimeConfig == nil {
		sylog.Fatalf("failed to initialize %s engine\n", name)
	}
}

// NewContainerLauncher will return a ContainerLauncher that uses the Engine named "name"
// and the config contained in "jsonConfig"
func NewContainerLauncher(name string, jsonConfig []byte) (launcher *ContainerLauncher, err error) {
	sylog.Debugf("Attempting to create ContainerLauncher using %s Engine\n", name)
	launcher, ok := registeredEngines[name]

	if !ok {
		sylog.Errorf("Runtime engine %s does not exist", name)
		return nil, fmt.Errorf("runtime engine %s does not exist", name)
	}

	if err := launcher.SetConfig(jsonConfig); err != nil {
		sylog.Errorf("Unable to set %s runtime config: %v\n", name, err)
		return nil, err
	}

	return launcher, nil
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
	registeredEngines = make(map[string]*ContainerLauncher)
	registeredEngineRPCMethods = make(map[string]interface{})

	// Registers engines there to compose another engine and inherit
	// parent engine methods, useful to override some Engine interface
	// methods to alter execution by keeping the core engine.
	// Since Singularity is the core, we must ensure it's the first
	// registered
	engine := &singularity.Engine{}
	methods := new(singularityRpcServer.Methods)
	registerEngine(engine, singularityConfig.Name)
	registerEngineRPCMethods(methods, singularityConfig.Name)
}

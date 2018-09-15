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
	"syscall"

	"github.com/singularityware/singularity/src/runtime/engines/config"
	"github.com/singularityware/singularity/src/runtime/engines/config/starter"
	"github.com/singularityware/singularity/src/runtime/engines/imgbuild"
	"github.com/singularityware/singularity/src/runtime/engines/singularity"
	singularityRpcServer "github.com/singularityware/singularity/src/runtime/engines/singularity/rpc/server"
)

// Engine is the combination of an EngineOperations and a config.Common. The singularity
// startup routines (src/runtime/startup/*) can spawn a container process from this type
type Engine struct {
	EngineOperations
	*config.Common
}

// EngineOperations is an interface describing necessary operations to launch a
// container process.
type EngineOperations interface {
	// Config returns the current EngineConfig, used to populate the Common struct
	Config() config.EngineConfig
	// InitConfig is responsible for storing the parse config.Common inside
	// the EngineOperations implementation.
	InitConfig(*config.Common)
	// PrepareConfig is called in stage1 to validate and prepare container configuration
	PrepareConfig(net.Conn, *starter.Config) error
	// CreateContainer is called in smaster and does mount operations, etc... to
	// set up the container environment for the payload proc
	CreateContainer(int, net.Conn) error
	// StartProcess is called in stage2 after waiting on RPC server exit. It is
	// responsible for exec'ing the payload proc in the container
	StartProcess(net.Conn) error
	// PostStartProcess is called in smaster after successful execution of container process
	PostStartProcess(int) error
	// MonitorContainer is called in smaster once the container proc has been spawned. It
	// will typically block until the container proc exists
	MonitorContainer(int) (syscall.WaitStatus, error)
	// CleanupContainer is called in smaster after the MontiorContainer returns. It is responsible
	// for ensuring that the container has been properly torn down
	CleanupContainer() error
}

// NewEngine returns the engine described by the JSON []byte configuration
func NewEngine(b []byte) (*Engine, error) {
	// Parse json []byte into map to first grab engineName value
	jsonMap := make(map[string]interface{})
	if err := json.Unmarshal(b, &jsonMap); err != nil {
		return nil, err
	}

	// Convert engineName from interface{} to string type
	if _, ok := jsonMap["engineName"]; !ok {
		return nil, fmt.Errorf("engineName field not found")
	}
	engineName := jsonMap["engineName"].(string)

	// Ensure engineName exists
	if _, ok := registeredEngineOperations[engineName]; !ok {
		return nil, fmt.Errorf("Engine name %s not found, failing", engineName)
	}

	// Create empty Engine object with properly initialized EngineConfig && EngineOperations
	e := &Engine{
		EngineOperations: registeredEngineOperations[engineName],
		Common: &config.Common{
			EngineConfig: registeredEngineOperations[engineName].Config(),
		},
	}

	// Now parse Common JSON configuration with EngineConfig associated
	// to corresponding engine
	if err := json.Unmarshal(b, e.Common); err != nil {
		return nil, fmt.Errorf("Unable to parse JSON into e.Common: %s", err)
	}
	e.InitConfig(e.Common)
	return e, nil
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

	// register singularity engine
	registerEngineOperations(&singularity.EngineOperations{EngineConfig: singularity.NewConfig()}, singularity.Name)
	// register imgbuild engine
	registerEngineOperations(&imgbuild.EngineOperations{EngineConfig: &imgbuild.EngineConfig{}}, imgbuild.Name)

	registeredEngineRPCMethods = make(map[string]interface{})

	// register singularity rpcmethods
	methods := new(singularityRpcServer.Methods)
	registerEngineRPCMethods(methods, singularity.Name)
	registerEngineRPCMethods(methods, imgbuild.Name)
}

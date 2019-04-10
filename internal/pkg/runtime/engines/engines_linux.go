// Copyright (c) 2019, Sylabs Inc. All rights reserved.
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
	imgbuildConfig "github.com/sylabs/singularity/internal/pkg/runtime/engines/imgbuild/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/oci"
	ociserver "github.com/sylabs/singularity/internal/pkg/runtime/engines/oci/rpc/server"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/rpc/server"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
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

// GetName returns the engine name set in JSON []byte configuration.
func GetName(b []byte) string {
	engineName := struct {
		EngineName string `json:"engineName"`
	}{}
	if err := json.Unmarshal(b, &engineName); err != nil {
		return ""
	}
	return engineName.EngineName
}

// NewEngine returns the engine described by the JSON []byte configuration.
func NewEngine(b []byte) (*Engine, error) {
	engineName := GetName(b)

	// ensure engine with given name is registered
	eOp, ok := registeredEngineOperations[engineName]
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
	registeredEngineOperations[singularityConfig.Name] = &singularity.EngineOperations{EngineConfig: singularityConfig.NewConfig()}
	registeredEngineOperations[imgbuildConfig.Name] = &imgbuild.EngineOperations{EngineConfig: &imgbuildConfig.EngineConfig{}}
	registeredEngineOperations[oci.Name] = &oci.EngineOperations{EngineConfig: &oci.EngineConfig{}}

	// register singularity rpc methods
	methods := new(server.Methods)
	registeredEngineRPCMethods = make(map[string]interface{})
	registeredEngineRPCMethods[singularityConfig.Name] = methods
	registeredEngineRPCMethods[imgbuildConfig.Name] = methods

	ocimethods := new(ociserver.Methods)
	ocimethods.Methods = methods
	registeredEngineRPCMethods[oci.Name] = ocimethods
}

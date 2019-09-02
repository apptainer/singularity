// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"github.com/sylabs/singularity/internal/pkg/runtime/engine"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config"
	ociServer "github.com/sylabs/singularity/internal/pkg/runtime/engine/oci/rpc/server"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/singularity/rpc/server"
)

// EngineOperations is a Singularity OCI runtime engine that implements engine.Operations.
// Basically, this is the core of `singularity oci` commands.
type EngineOperations struct {
	CommonConfig *config.Common `json:"-"`
	EngineConfig *EngineConfig  `json:"engineConfig"`
}

// InitConfig stores the parsed config.Common inside the engine.
//
// Since this method simply stores config.Common, it does not matter
// whether or not there are any elevated privileges during this call.
//
// Most likely this still will be executed as root since `singularity oci`
// command set requires privileged execution.
func (e *EngineOperations) InitConfig(cfg *config.Common) {
	e.CommonConfig = cfg
}

// Config returns a pointer to a singularity.EngineConfig literal as a
// config.EngineConfig interface. This pointer gets stored in the Engine.Common
// field.

// Config returns a pointer to EngineConfig literal as a config.EngineConfig
// interface. This pointer gets stored in the Engine.Common field.
//
// Since this method simply returns a zero value of the concrete
// EngineConfig, it does not matter whether or not there are any elevated
// privileges during this call. However, most likely this still will be executed
// as root since `singularity oci` command set requires privileged execution.
func (e *EngineOperations) Config() config.EngineConfig {
	return e.EngineConfig
}

func init() {
	engine.RegisterOperations(
		Name,
		&EngineOperations{
			EngineConfig: &EngineConfig{},
		},
	)

	ocimethods := new(ociServer.Methods)
	ocimethods.Methods = new(server.Methods)
	engine.RegisterRPCMethods(
		Name,
		ocimethods,
	)
}

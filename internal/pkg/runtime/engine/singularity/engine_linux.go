// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"github.com/sylabs/singularity/internal/pkg/runtime/engine"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/singularity/rpc/server"
	"github.com/sylabs/singularity/pkg/runtime/engine/config"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engine/singularity/config"
)

// EngineOperations is a Singularity runtime engine that implements engine.Operations.
// Basically, this is the core of `singularity run/exec/shell/instance` commands.
type EngineOperations struct {
	CommonConfig *config.Common                  `json:"-"`
	EngineConfig *singularityConfig.EngineConfig `json:"engineConfig"`
}

// InitConfig stores the parsed config.Common inside the engine.
//
// Since this method simply stores config.Common, it does not matter
// whether or not there are any elevated privileges during this call.
func (e *EngineOperations) InitConfig(cfg *config.Common) {
	e.CommonConfig = cfg
}

// Config returns a pointer to a singularityConfig.EngineConfig
// literal as a config.EngineConfig interface. This pointer
// gets stored in the engine.Engine.Common field.
//
// Since this method simply returns a zero value of the concrete
// EngineConfig, it does not matter whether or not there are any elevated
// privileges during this call.
func (e *EngineOperations) Config() config.EngineConfig {
	return e.EngineConfig
}

func init() {
	engine.RegisterOperations(
		singularityConfig.Name,
		&EngineOperations{
			EngineConfig: singularityConfig.NewConfig(),
		},
	)

	engine.RegisterRPCMethods(
		singularityConfig.Name,
		new(server.Methods),
	)
}

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
)

// EngineOperations implements the engines.EngineOperations interface for
// the image build process
type EngineOperations struct {
	CommonConfig *config.Common `json:"-"`
	EngineConfig *EngineConfig  `json:"engineConfig"`
}

// InitConfig initializes engines config internals
func (e *EngineOperations) InitConfig(cfg *config.Common) {
	e.CommonConfig = cfg
}

// Config returns the EngineConfig
func (e *EngineOperations) Config() config.EngineConfig {
	return e.EngineConfig
}

// CheckConfig validates EngineConfig setup
func (e *EngineOperations) CheckConfig() error {
	e.CommonConfig.OciConfig.SetProcessNoNewPrivileges(true)
	return nil
}

// IsRunAsInstance returns false
func (e *EngineOperations) IsRunAsInstance() bool {
	return false
}

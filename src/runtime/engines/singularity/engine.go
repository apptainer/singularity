// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"encoding/json"

	"github.com/singularityware/singularity/src/pkg/sylog"
	config "github.com/singularityware/singularity/src/runtime/engines/common/config"
	singularityConfig "github.com/singularityware/singularity/src/runtime/engines/singularity/config"
)

// EngineOperations describes a runtime engine
type EngineOperations struct {
	CommonConfig *config.Common                  `json:"-"`
	EngineConfig *singularityConfig.EngineConfig `json:"engineConfig"`
}

// InitConfig stores the pointer to config.Common
func (e *EngineOperations) InitConfig(cfg *config.Common) {
	e.CommonConfig = cfg

	if err := json.Unmarshal(cfg.EngineConfig, e.EngineConfig); err != nil {
		sylog.Fatalf("Unable to initialze Singularity engine config: %s\n", err)
	}

}

// IsRunAsInstance returns true if the runtime engine was run as an instance
func (engine *EngineOperations) IsRunAsInstance() bool {
	return false
}

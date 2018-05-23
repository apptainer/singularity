// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package runtime

import (
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	config "github.com/singularityware/singularity/src/pkg/workflows/config"
	oci "github.com/singularityware/singularity/src/pkg/workflows/oci/config"
	singularityConfig "github.com/singularityware/singularity/src/runtime/workflows/workflows/singularity/config"
)

// Engine describes a runtime engine
type Engine struct {
	singularityConfig.RuntimeEngineConfig
}

// InitConfig initializes a runtime configuration
func (engine *Engine) InitConfig() *config.RuntimeConfig {
	if engine.FileConfig == nil {
		engine.FileConfig = &singularityConfig.Configuration{}
		if err := config.Parser(buildcfg.SYSCONFDIR+"/singularity/singularity.conf", engine.FileConfig); err != nil {
			return nil
		}
	}
	cfg := &engine.RuntimeConfig
	cfg.RuntimeEngineSpec = &engine.RuntimeEngineSpec
	oci.DefaultRuntimeOciConfig(&cfg.OciConfig)
	return cfg
}

// IsRunAsInstance returns true if the runtime engine was run as an instance
func (engine *Engine) IsRunAsInstance() bool {
	return engine.GetInstance()
}

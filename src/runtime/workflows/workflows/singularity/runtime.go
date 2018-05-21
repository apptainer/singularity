// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package runtime

import (
	config "github.com/singularityware/singularity/src/pkg/workflows/config"
	oci "github.com/singularityware/singularity/src/pkg/workflows/oci/config"
	singularityConfig "github.com/singularityware/singularity/src/runtime/workflows/workflows/singularity/config"
)

// Engine describes a runtime engine
type Engine struct {
	singularityConfig.RuntimeEngineConfig
}

// InitConfig initializes a runtime configuration
func (e *Engine) InitConfig() *config.RuntimeConfig {
	if e.FileConfig == nil {
		e.FileConfig = &singularityConfig.Configuration{}
		if err := config.Parser("/usr/local/etc/singularity/singularity.conf", e.FileConfig); err != nil {
			return nil
		}
	}
	cfg := &e.RuntimeConfig
	oci.DefaultRuntimeOciConfig(&cfg.OciConfig)
	return cfg
}

// IsRunAsInstance returns true if the runtime engine was run as an instance
func (e *Engine) IsRunAsInstance() bool {
	return false
}
